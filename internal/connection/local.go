package connection

import (
	"io/fs"
	"os"
	"path/filepath"
)

type LocalFS struct{}

func NewLocalFS() *LocalFS {
	return &LocalFS{}
}

func (l *LocalFS) ReadFile(path string) ([]byte, error) {
	clean := filepath.Clean(path)
	return os.ReadFile(clean)
}

func (l *LocalFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	clean := filepath.Clean(path)
	dir := filepath.Dir(clean)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(clean, data, perm)
}

func (l *LocalFS) Stat(path string) (*FileInfo, error) {
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
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

func (l *LocalFS) Remove(path string) error {
	clean := filepath.Clean(path)
	return os.Remove(clean)
}

func (l *LocalFS) ReadDir(path string) ([]DirEntry, error) {
	clean := filepath.Clean(path)
	entries, err := os.ReadDir(clean)
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

func (l *LocalFS) HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return home, nil
}

func (l *LocalFS) MkdirAll(path string, perm fs.FileMode) error {
	clean := filepath.Clean(path)
	return os.MkdirAll(clean, perm)
}

func (l *LocalFS) GetEnv(key string) string {
	return os.Getenv(key)
}

func (l *LocalFS) Close() error {
	return nil
}
