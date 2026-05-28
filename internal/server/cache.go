package server

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	defaultMaxCacheGB = 50
	evictionInterval  = 5 * time.Minute
)

type CacheEvictor struct {
	thumbnailsDir string
	maxBytes      int64
}

func NewCacheEvictor(thumbnailsDir string) *CacheEvictor {
	return &CacheEvictor{
		thumbnailsDir: thumbnailsDir,
		maxBytes:      defaultMaxCacheGB * 1024 * 1024 * 1024,
	}
}

func (c *CacheEvictor) Start() {
	go func() {
		for {
			time.Sleep(evictionInterval)
			if err := c.evictIfNeeded(); err != nil {
				slog.Warn("cache eviction error", "error", err)
			}
		}
	}()
}

type fileInfo struct {
	path    string
	modTime time.Time
	size    int64
}

func (c *CacheEvictor) evictIfNeeded() error {
	var totalSize int64
	var files []fileInfo

	err := filepath.Walk(c.thumbnailsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		totalSize += info.Size()
		files = append(files, fileInfo{path: path, modTime: info.ModTime(), size: info.Size()})
		return nil
	})
	if err != nil {
		return err
	}

	if totalSize <= c.maxBytes {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	freed := int64(0)
	for _, f := range files {
		if totalSize-freed <= c.maxBytes {
			break
		}
		if err := os.Remove(f.path); err == nil {
			freed += f.size
		}
	}

	if freed > 0 {
		slog.Info("cache eviction completed", "freed_bytes", freed)
	}

	var mu sync.Mutex
	err = filepath.Walk(c.thumbnailsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if path == c.thumbnailsDir {
			return nil
		}
		mu.Lock()
		defer mu.Unlock()
		entries, _ := os.ReadDir(path)
		if len(entries) == 0 {
			os.Remove(path)
		}
		return nil
	})

	return err
}
