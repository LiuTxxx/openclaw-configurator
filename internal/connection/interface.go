package connection

import (
	"io/fs"
	"time"
)

type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	Mode    fs.FileMode `json:"-"`
	IsDir   bool      `json:"isDir"`
	ModTime time.Time `json:"modTime"`
}

type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
}

type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	Stat(path string) (*FileInfo, error)
	Remove(path string) error
	ReadDir(path string) ([]DirEntry, error)
	HomeDir() (string, error)
	MkdirAll(path string, perm fs.FileMode) error
	GetEnv(key string) string
	Close() error
}
