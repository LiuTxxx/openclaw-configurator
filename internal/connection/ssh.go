package connection

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	AuthType   string `json:"authType"` // "password" or "key"
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"` // raw PEM content (backward compat)
	KeyPath    string `json:"keyPath,omitempty"`    // path to key file on LOCAL machine
}

type SSHFS struct {
	client     *ssh.Client
	sftpClient *sftp.Client
	user       string
}

func NewSSHFS(cfg SSHConfig) (*SSHFS, error) {
	if cfg.Host == "" || cfg.User == "" {
		return nil, errors.New("host and user are required")
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		cfg.Port = 22
	}

	var authMethods []ssh.AuthMethod
	switch cfg.AuthType {
	case "password":
		if cfg.Password == "" {
			return nil, errors.New("password is required for password auth")
		}
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	case "key":
		keyData, err := resolvePrivateKey(cfg)
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", cfg.AuthType)
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	log.Println("[WARN] SSH host key verification is disabled — connection is not protected against MITM attacks")

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ssh connection failed: %w", err)
	}

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("sftp session failed: %w", err)
	}

	return &SSHFS{
		client:     client,
		sftpClient: sftpClient,
		user:       cfg.User,
	}, nil
}

func resolvePrivateKey(cfg SSHConfig) ([]byte, error) {
	if cfg.PrivateKey != "" {
		return []byte(cfg.PrivateKey), nil
	}

	keyPath := cfg.KeyPath
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home dir for default key: %w", err)
		}
		keyPath = filepath.Join(home, ".ssh", "id_rsa")
	}

	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %q: %w", keyPath, err)
	}
	return data, nil
}

func (s *SSHFS) ReadFile(filePath string) ([]byte, error) {
	clean := path.Clean(filePath)
	f, err := s.sftpClient.Open(clean)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (s *SSHFS) WriteFile(filePath string, data []byte, perm fs.FileMode) error {
	clean := path.Clean(filePath)
	f, err := s.sftpClient.OpenFile(clean, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(perm); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func (s *SSHFS) Stat(filePath string) (*FileInfo, error) {
	clean := path.Clean(filePath)
	info, err := s.sftpClient.Stat(clean)
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Name:    info.Name(),
		Size:    info.Size(),
		Mode:    info.Mode(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime(),
	}, nil
}

func (s *SSHFS) Remove(filePath string) error {
	clean := path.Clean(filePath)
	return s.sftpClient.Remove(clean)
}

func (s *SSHFS) ReadDir(dirPath string) ([]DirEntry, error) {
	clean := path.Clean(dirPath)
	entries, err := s.sftpClient.ReadDir(clean)
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, DirEntry{
			Name:  e.Name(),
			IsDir: e.IsDir(),
		})
	}
	return result, nil
}

func (s *SSHFS) HomeDir() (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	out, err := session.Output("echo $HOME")
	if err != nil {
		return "/home/" + s.user, nil
	}
	home := string(out)
	if len(home) > 0 && home[len(home)-1] == '\n' {
		home = home[:len(home)-1]
	}
	if home == "" {
		return "/home/" + s.user, nil
	}
	return home, nil
}

func (s *SSHFS) MkdirAll(dirPath string, perm fs.FileMode) error {
	clean := path.Clean(dirPath)
	return s.sftpClient.MkdirAll(clean)
}

func (s *SSHFS) GetEnv(key string) string {
	if !envVarNameRe.MatchString(key) {
		return ""
	}
	session, err := s.client.NewSession()
	if err != nil {
		return ""
	}
	defer session.Close()
	out, err := session.Output("printenv " + key)
	if err != nil {
		return ""
	}
	val := string(out)
	if len(val) > 0 && val[len(val)-1] == '\n' {
		val = val[:len(val)-1]
	}
	return val
}

func (s *SSHFS) Close() error {
	var errs []error
	if s.sftpClient != nil {
		if err := s.sftpClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
