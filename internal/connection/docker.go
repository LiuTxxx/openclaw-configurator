package connection

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var containerNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
var envVarNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type DockerConfig struct {
	Container string `json:"container"`
}

type DockerFS struct {
	container string
}

func NewDockerFS(cfg DockerConfig) (*DockerFS, error) {
	if cfg.Container == "" {
		return nil, errors.New("container name or ID is required")
	}
	if !containerNameRe.MatchString(cfg.Container) && !isContainerID(cfg.Container) {
		return nil, fmt.Errorf("invalid container name or ID")
	}

	if err := checkDockerCLI(); err != nil {
		return nil, err
	}

	out, err := dockerExec(cfg.Container, "echo", "ok")
	if err != nil {
		return nil, fmt.Errorf("cannot reach container: %w", err)
	}
	if strings.TrimSpace(out) != "ok" {
		return nil, fmt.Errorf("unexpected response from container")
	}

	return &DockerFS{container: cfg.Container}, nil
}

func isContainerID(s string) bool {
	if len(s) < 12 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func checkDockerCLI() error {
	_, err := exec.LookPath("docker")
	if err != nil {
		return errors.New("docker CLI not found on PATH — install Docker or Docker Desktop first")
	}
	return nil
}

func dockerExec(container string, args ...string) (string, error) {
	cmdArgs := append([]string{"exec", "--", container}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}

func dockerExecStdin(container string, stdin []byte, args ...string) error {
	cmdArgs := append([]string{"exec", "-i", "--", container}, args...)
	cmd := exec.Command("docker", cmdArgs...)
	cmd.Stdin = bytes.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

func (d *DockerFS) ReadFile(p string) ([]byte, error) {
	out, err := dockerExec(d.container, "cat", "--", p)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

func (d *DockerFS) WriteFile(p string, data []byte, perm fs.FileMode) error {
	if len(data) > 512*1024 {
		return fmt.Errorf("file too large for docker write (max 512KB)")
	}
	script := fmt.Sprintf("cat > %s && chmod %o %s", shellQuote(p), perm, shellQuote(p))
	return dockerExecStdin(d.container, data, "sh", "-c", script)
}

func (d *DockerFS) Stat(p string) (*FileInfo, error) {
	qp := shellQuote(p)
	// Pipe-delimited plain text avoids JSON injection from filenames
	// %s=size, %Y=mtime epoch, %F=file type — works on GNU coreutils and BusyBox
	script := fmt.Sprintf(`stat -c '%%s|%%Y|%%F' %s 2>/dev/null || echo _FALLBACK_`, qp)
	out, err := dockerExec(d.container, "sh", "-c", script)
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)

	if out != "_FALLBACK_" {
		parts := strings.SplitN(out, "|", 3)
		if len(parts) == 3 {
			var size, mtime int64
			fmt.Sscanf(parts[0], "%d", &size)
			fmt.Sscanf(parts[1], "%d", &mtime)
			isDir := strings.Contains(strings.ToLower(parts[2]), "directory")
			return &FileInfo{
				Name:    pathBase(p),
				Size:    size,
				IsDir:   isDir,
				ModTime: time.Unix(mtime, 0),
			}, nil
		}
	}

	testScript := fmt.Sprintf(`test -d %s && echo DIR || echo FILE`, qp)
	testOut, testErr := dockerExec(d.container, "sh", "-c", testScript)
	if testErr != nil {
		return nil, fmt.Errorf("cannot stat path")
	}
	return &FileInfo{
		Name:  pathBase(p),
		IsDir: strings.TrimSpace(testOut) == "DIR",
	}, nil
}

func (d *DockerFS) Remove(p string) error {
	_, err := dockerExec(d.container, "rm", "-f", "--", p)
	return err
}

func (d *DockerFS) ReadDir(dirPath string) ([]DirEntry, error) {
	qp := shellQuote(dirPath)
	out, err := dockerExec(d.container, "sh", "-c", fmt.Sprintf("ls -1ap %s", qp))
	if err != nil {
		return nil, err
	}
	var entries []DirEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "./" || line == "../" || line == "." || line == ".." {
			continue
		}
		isDir := strings.HasSuffix(line, "/")
		name := strings.TrimSuffix(line, "/")
		if name == "" {
			continue
		}
		entries = append(entries, DirEntry{
			Name:  name,
			IsDir: isDir,
		})
	}
	return entries, nil
}

func (d *DockerFS) HomeDir() (string, error) {
	out, err := dockerExec(d.container, "sh", "-c", "echo $HOME")
	if err == nil {
		home := strings.TrimSpace(out)
		if home != "" {
			return home, nil
		}
	}
	out, err = dockerExec(d.container, "sh", "-c",
		`getent passwd "$(whoami)" 2>/dev/null | cut -d: -f6 || echo /root`)
	if err != nil {
		return "/root", nil
	}
	home := strings.TrimSpace(out)
	if home == "" {
		return "/root", nil
	}
	return home, nil
}

func (d *DockerFS) MkdirAll(p string, perm fs.FileMode) error {
	_, err := dockerExec(d.container, "mkdir", "-p", "--", p)
	return err
}

func (d *DockerFS) GetEnv(key string) string {
	if !envVarNameRe.MatchString(key) {
		return ""
	}
	out, err := dockerExec(d.container, "sh", "-c", "printenv "+shellQuote(key))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func (d *DockerFS) Close() error {
	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func pathBase(p string) string {
	p = strings.TrimRight(p, "/\\")
	if p == "" {
		return "."
	}
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}
