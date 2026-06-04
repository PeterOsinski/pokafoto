package service

import (
	"io"
	"os"

	"github.com/minio/minio-go/v7"
)

type StorageProvider interface {
	PutObject(key string, filePath string) error
	GetObject(key string, destPath string) error
	GetObjectStream(key string) (io.ReadCloser, error)
	IsConnected() bool
	DeleteObject(key string) error
	ListObjects(prefix string) ([]string, error)
	Client() *minio.Client
}

type FileSystem interface {
	ReadDir(name string) ([]os.DirEntry, error)
	Remove(name string) error
	RemoveAll(path string) error
	Stat(name string) (os.FileInfo, error)
	Statfs(path string) (TotalBlocks, BlockSize, FreeBlocks uint64)
	CreateTemp(dir, pattern string) (*os.File, error)
	MkdirAll(path string, perm os.FileMode) error
	Walk(root string, fn func(path string, info os.FileInfo, err error) error) error
	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
	ReadFile(name string) ([]byte, error)
}

type TotalBlocks = uint64
type BlockSize = uint64
type FreeBlocks = uint64
