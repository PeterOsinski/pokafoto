package service

import (
	"os"
	"testing"

	"github.com/drive/drive/internal/config"
)

func TestCacheEvictor_New_shouldInitializeFields(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	fs := NewMockFS()
	ce := NewCacheEvictor(cfg, fs, nil)

	if ce.cfg != cfg {
		t.Error("cfg not set")
	}
	if ce.fs != fs {
		t.Error("fs not set")
	}
	if ce.thumbnailsDir != "/data/thumbnails" {
		t.Errorf("expected /data/thumbnails, got %s", ce.thumbnailsDir)
	}
	if ce.originalsDir != "/data/originals" {
		t.Errorf("expected /data/originals, got %s", ce.originalsDir)
	}
	if ce.storagePath != "/data" {
		t.Errorf("expected /data, got %s", ce.storagePath)
	}
}

func TestCacheEvictor_ComputeMaxBytes_setsFloorAt1GB(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"
	cfg.Storage.MaxDiskUsagePct = 1

	fs := NewMockFS()
	ce := NewCacheEvictor(cfg, fs, nil)

	maxBytes := ce.ComputeMaxBytes()

	if maxBytes < 1*1024*1024*1024 {
		t.Errorf("expected at least 1GB floor, got %d bytes", maxBytes)
	}
}

func TestCacheEvictor_ComputeMaxBytes_subtractsOriginals(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"
	cfg.Storage.MaxDiskUsagePct = 1

	fs := NewMockFS()
	fs.MkdirAll("/data/originals", 0755)

	ce := NewCacheEvictor(cfg, fs, nil)
	maxBytesWithZero := ce.ComputeMaxBytes()

	fs.AddFile("/data/originals/user1/video.mp4", make([]byte, 500*1024*1024))
	maxBytesWithFile := ce.ComputeMaxBytes()

	if maxBytesWithFile >= maxBytesWithZero {
		t.Errorf("expected maxBytes to decrease when originals added, got %d (with file) >= %d (without)", maxBytesWithFile, maxBytesWithZero)
	}
}

func TestCacheEvictor_calculateMaxBytes_usesComputedValue(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"
	cfg.Storage.MaxDiskUsagePct = 1

	fs := NewMockFS()
	ce := NewCacheEvictor(cfg, fs, nil)

	ce.calculateMaxBytes()

	if ce.maxBytes <= 0 {
		t.Error("expected non-zero maxBytes")
	}
}

func TestCacheEvictor_evictIfNeeded_noThumbnails_shouldSucceed(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	fs := NewMockFS()
	ce := NewCacheEvictor(cfg, fs, nil)
	ce.maxBytes = 100 * 1024 * 1024 * 1024

	err := ce.evictIfNeeded()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCacheEvictor_evictIfNeeded_belowMax_noEviction(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	fs := NewMockFS()
	fs.AddFile("/data/thumbnails/file1/large", make([]byte, 1024))
	fs.AddFile("/data/thumbnails/file2/small", make([]byte, 512))

	ce := NewCacheEvictor(cfg, fs, nil)
	ce.maxBytes = 100 * 1024 * 1024 * 1024

	err := ce.evictIfNeeded()
	if err != nil {
		t.Fatal(err)
	}

	_, err = fs.Stat("/data/thumbnails/file1/large")
	if err != nil {
		t.Error("file1/large should not have been evicted")
	}
	_, err = fs.Stat("/data/thumbnails/file2/small")
	if err != nil {
		t.Error("file2/small should not have been evicted")
	}
}

func TestCacheEvictor_evictIfNeeded_aboveMax_evictsFiles(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	fs := NewMockFS()
	fs.AddFile("/data/thumbnails/file1/large", make([]byte, 1024*1024*1024))
	fs.AddFile("/data/thumbnails/file1/small", make([]byte, 100*1024*1024))
	fs.AddFile("/data/thumbnails/file2/preview", make([]byte, 500*1024*1024))

	ce := NewCacheEvictor(cfg, fs, nil)
	ce.maxBytes = 500 * 1024 * 1024

	err := ce.evictIfNeeded()
	if err != nil {
		t.Fatal(err)
	}

	remainingTotal := int64(0)
	fs.Walk("/data/thumbnails", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			remainingTotal += info.Size()
		}
		return nil
	})

	if remainingTotal > ce.maxBytes {
		t.Errorf("expected remaining %d <= maxBytes %d", remainingTotal, ce.maxBytes)
	}
}

func TestCacheEvictor_evictIfNeeded_aggressiveMode_targetsSeventyPercent(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	fs := NewMockFS()
	for i := 0; i < 10; i++ {
		fs.AddFile("/data/thumbnails/dir"+string(rune('a'+i))+"/large.webp", make([]byte, 300*1024*1024))
	}

	ce := NewCacheEvictor(cfg, fs, nil)
	ce.maxBytes = 500 * 1024 * 1024

	err := ce.evictIfNeeded()
	if err != nil {
		t.Fatal(err)
	}

	remainingTotal := int64(0)
	fs.Walk("/data/thumbnails", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			remainingTotal += info.Size()
		}
		return nil
	})

	aggressiveTarget := ce.maxBytes * 70 / 100
	if remainingTotal > aggressiveTarget {
		t.Errorf("aggressive mode: expected remaining %d <= 70%% target %d", remainingTotal, aggressiveTarget)
	}
}

func TestCacheEvictor_evictIfNeeded_cleansEmptyDirectories(t *testing.T) {
	t.Parallel()
	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	fs := NewMockFS()
	fs.AddFile("/data/thumbnails/empty-dir/file.jpg", make([]byte, 200*1024*1024))

	ce := NewCacheEvictor(cfg, fs, nil)
	ce.maxBytes = 50 * 1024 * 1024

	err := ce.evictIfNeeded()
	if err != nil {
		t.Fatal(err)
	}

	entries, _ := fs.ReadDir("/data/thumbnails")
	if len(entries) > 0 {
		for _, entry := range entries {
			e2, _ := fs.ReadDir("/data/thumbnails/" + entry.Name())
			if len(e2) == 0 {
				t.Errorf("empty directory %s should have been cleaned up", entry.Name())
			}
		}
	}
}
