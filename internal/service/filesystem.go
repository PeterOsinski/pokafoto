package service

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type RealFS struct{}

func NewRealFS() *RealFS {
	return &RealFS{}
}

func (fs *RealFS) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (fs *RealFS) Remove(name string) error {
	return os.Remove(name)
}

func (fs *RealFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (fs *RealFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *RealFS) Statfs(path string) (uint64, uint64, uint64) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	return total, uint64(stat.Bsize), free
}

func (fs *RealFS) CreateTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}

func (fs *RealFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *RealFS) Walk(root string, fn func(path string, info os.FileInfo, err error) error) error {
	return filepath.Walk(root, fn)
}

func (fs *RealFS) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (fs *RealFS) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (fs *RealFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
