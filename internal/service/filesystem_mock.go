package service

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type memFileInfo struct {
	name    string
	size    int64
	isDir   bool
	modTime time.Time
}

func (m *memFileInfo) Name() string       { return m.name }
func (m *memFileInfo) Size() int64        { return m.size }
func (m *memFileInfo) Mode() os.FileMode  { return 0o644 }
func (m *memFileInfo) ModTime() time.Time { return m.modTime }
func (m *memFileInfo) IsDir() bool        { return m.isDir }
func (m *memFileInfo) Sys() interface{}   { return nil }

type memDirEntry struct {
	info *memFileInfo
}

func (m *memDirEntry) Name() string               { return m.info.Name() }
func (m *memDirEntry) IsDir() bool                 { return m.info.IsDir() }
func (m *memDirEntry) Type() fs.FileMode           { return m.info.Mode().Type() }
func (m *memDirEntry) Info() (fs.FileInfo, error)  { return m.info, nil }

type MockFS struct {
	mu      sync.RWMutex
	files   map[string][]byte
	dirs    map[string]*memFileInfo
	removed map[string]bool
}

func NewMockFS() *MockFS {
	return &MockFS{
		files:   make(map[string][]byte),
		dirs:    make(map[string]*memFileInfo),
		removed: make(map[string]bool),
	}
}

func (m *MockFS) ReadDir(name string) ([]os.DirEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.removed[name] {
		return nil, fmt.Errorf("open %s: no such file or directory", name)
	}

	prefix := name
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var entries []os.DirEntry
	seen := make(map[string]bool)
	for path, info := range m.files {
		if strings.HasPrefix(path, prefix) {
			rest := strings.TrimPrefix(path, prefix)
			parts := strings.SplitN(rest, "/", 2)
			if !seen[parts[0]] {
				seen[parts[0]] = true
				entries = append(entries, &memDirEntry{info: &memFileInfo{
					name:  parts[0],
					size:  int64(len(info)),
					isDir: len(parts) > 1,
				}})
			}
		}
	}
	for path, info := range m.dirs {
		if path != name && strings.HasPrefix(path, prefix) {
			rest := strings.TrimPrefix(path, prefix)
			if !strings.Contains(rest, "/") && !seen[rest] {
				seen[rest] = true
				entries = append(entries, &memDirEntry{info: info})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (m *MockFS) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, name)
	m.removed[name] = true
	return nil
}

func (m *MockFS) RemoveAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := path
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	for p := range m.files {
		if p == path || strings.HasPrefix(p, prefix) {
			delete(m.files, p)
		}
	}
	for p := range m.dirs {
		if p == path || strings.HasPrefix(p, prefix) {
			delete(m.dirs, p)
		}
	}
	m.removed[path] = true
	return nil
}

func (m *MockFS) Stat(name string) (os.FileInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.removed[name] {
		return nil, fmt.Errorf("stat %s: no such file or directory", name)
	}

	if data, ok := m.files[name]; ok {
		return &memFileInfo{name: filepath.Base(name), size: int64(len(data)), modTime: time.Now()}, nil
	}
	if info, ok := m.dirs[name]; ok {
		return info, nil
	}
	return nil, fmt.Errorf("stat %s: no such file or directory", name)
}

func (m *MockFS) Statfs(path string) (uint64, uint64, uint64) {
	return 100 * 1024 * 1024 * 1024, 4096, 50 * 1024 * 1024 * 1024
}

func (m *MockFS) CreateTemp(dir, pattern string) (*os.File, error) {
	return nil, fmt.Errorf("mock CreateTemp not implemented")
}

func (m *MockFS) MkdirAll(path string, perm os.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = &memFileInfo{name: filepath.Base(path), isDir: true, modTime: time.Now()}
	return nil
}

func (m *MockFS) Walk(root string, fn func(path string, info os.FileInfo, err error) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for p, data := range m.files {
		if p == root || strings.HasPrefix(p, root+"/") {
			if !m.removed[p] {
				if err := fn(p, &memFileInfo{name: filepath.Base(p), size: int64(len(data)), modTime: time.Now()}, nil); err != nil {
					return err
				}
			}
		}
	}
	for p, info := range m.dirs {
		if (p == root || strings.HasPrefix(p, root+"/")) && p != root {
			if err := fn(p, info, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *MockFS) Open(name string) (*os.File, error) {
	return nil, fmt.Errorf("mock Open not implemented")
}

func (m *MockFS) Create(name string) (*os.File, error) {
	return nil, fmt.Errorf("mock Create not implemented")
}

func (m *MockFS) ReadFile(name string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.removed[name] {
		return nil, fmt.Errorf("read %s: no such file or directory", name)
	}

	if data, ok := m.files[name]; ok {
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	}
	return nil, fmt.Errorf("read %s: no such file or directory", name)
}

func (m *MockFS) AddFile(path string, content []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = content
	dir := filepath.Dir(path)
	for dir != "." && dir != "/" {
		if _, exists := m.dirs[dir]; !exists {
			m.dirs[dir] = &memFileInfo{name: filepath.Base(dir), isDir: true, modTime: time.Now()}
		}
		dir = filepath.Dir(dir)
	}
}
