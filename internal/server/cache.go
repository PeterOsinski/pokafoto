package server

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/drive/drive/internal/config"
	"golang.org/x/sys/unix"
)

const evictionInterval = 5 * time.Minute

type CacheEvictor struct {
	cfg          *config.Config
	thumbnailsDir string
	maxBytes      int64
}

func NewCacheEvictor(cfg *config.Config) *CacheEvictor {
	return &CacheEvictor{
		cfg:           cfg,
		thumbnailsDir: cfg.ThumbnailsDir(),
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

func (c *CacheEvictor) ComputeMaxBytes() int64 {
	var stat unix.Statfs_t
	if err := unix.Statfs(c.cfg.Storage.Local.Path, &stat); err != nil {
		return 50 * 1024 * 1024 * 1024
	}
	total := stat.Blocks * uint64(stat.Bsize)
	maxAllowed := int64(total) * int64(c.cfg.MaxDiskUsagePercent()) / 100

	var originalsSize int64
	filepath.Walk(c.cfg.OriginalsDir(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			originalsSize += info.Size()
		}
		return nil
	})

	maxThumbnail := maxAllowed - originalsSize - 2*1024*1024*1024
	if maxThumbnail < 1*1024*1024*1024 {
		maxThumbnail = 1 * 1024 * 1024 * 1024
	}
	return maxThumbnail
}

func (c *CacheEvictor) evictIfNeeded() error {
	c.calculateMaxBytes()
	maxBytes := c.maxBytes

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

	if totalSize <= maxBytes {
		return nil
	}

	aggressiveThreshold := maxBytes * 80 / 100
	aggressive := totalSize > maxBytes+aggressiveThreshold

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	freed := int64(0)
	target := maxBytes
	if aggressive {
		target = maxBytes * 70 / 100
		slog.Warn("cache exceeding aggressive threshold, evicting more", "total_bytes", totalSize, "max_bytes", maxBytes)
	}

	for _, f := range files {
		if totalSize-freed <= target {
			break
		}
		if err := os.Remove(f.path); err == nil {
			freed += f.size
		}
	}

	if freed > 0 {
		slog.Info("cache eviction completed", "freed_bytes", freed)
		if totalSize > maxBytes {
			slog.Warn("cache still over limit after eviction", "total", totalSize, "max", maxBytes)
		}
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

func (c *CacheEvictor) calculateMaxBytes() {
	c.maxBytes = c.ComputeMaxBytes()
	if c.maxBytes <= 0 {
		c.maxBytes = 50 * 1024 * 1024 * 1024
	}
}
