package service

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/drive/drive/internal/model"
)

func createTestJPEG(t *testing.T, dir string, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for x := 0; x < 800; x++ {
		for y := 0; y < 600; y++ {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 100, 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test jpeg: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode test jpeg: %v", err)
	}
	return path
}

func TestThumbnailService_GeneratePreview_shouldBeWebP(t *testing.T) {
	dir := t.TempDir()
	src := createTestJPEG(t, dir, "test.jpg")

	ts := NewThumbnailService(dir)
	thumbs, err := ts.GenerateAll("test-file-id", src, "image/jpeg")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	var preview *model.Thumbnail
	for _, th := range thumbs {
		if th.Size == model.ThumbSizePreview {
			preview = th
			break
		}
	}
	if preview == nil {
		t.Fatal("expected preview thumbnail, got nil")
	}

	if preview.Format != "webp" {
		t.Errorf("expected format webp, got %s", preview.Format)
	}

	data, err := os.ReadFile(preview.LocalPath)
	if err != nil {
		t.Fatalf("read preview file: %v", err)
	}

	if len(data) < 12 {
		t.Fatal("file too small for RIFF header")
	}

	if string(data[0:4]) != "RIFF" {
		t.Errorf("expected RIFF header, got %s", string(data[0:4]))
	}
	if string(data[8:12]) != "WEBP" {
		t.Errorf("expected WEBP FourCC, got %s", string(data[8:12]))
	}
}

func TestThumbnailService_GenerateSmall_shouldBeJPEG(t *testing.T) {
	dir := t.TempDir()
	src := createTestJPEG(t, dir, "test.jpg")

	ts := NewThumbnailService(dir)
	thumbs, err := ts.GenerateAll("test-file-id", src, "image/jpeg")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	for _, th := range thumbs {
		if th.Size == model.ThumbSizeSmall {
			data, err := os.ReadFile(th.LocalPath)
			if err != nil {
				t.Fatalf("read small thumb: %v", err)
			}
			if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
				t.Errorf("expected JPEG header (FF D8), got %X %X", data[0], data[1])
			}
			return
		}
	}
	t.Fatal("expected small thumbnail, not found")
}

func TestThumbnailService_GenerateLarge_shouldBe300pxJPEG(t *testing.T) {
	dir := t.TempDir()
	src := createTestJPEG(t, dir, "test.jpg")

	ts := NewThumbnailService(dir)
	thumbs, err := ts.GenerateAll("test-file-id", src, "image/jpeg")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	for _, th := range thumbs {
		if th.Size == model.ThumbSizeLarge {
			if th.Width != 300 {
				t.Errorf("expected width 300, got %d", th.Width)
			}
			if th.Format != "jpeg" {
				t.Errorf("expected format jpeg, got %s", th.Format)
			}
			data, err := os.ReadFile(th.LocalPath)
			if err != nil {
				t.Fatalf("read large thumb: %v", err)
			}
			if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
				t.Errorf("expected JPEG header (FF D8), got %X %X", data[0], data[1])
			}
			return
		}
	}
	t.Fatal("expected large thumbnail, not found")
}

func TestThumbnailService_GenerateAll_shouldFlushAllFilesToDisk(t *testing.T) {
	dir := t.TempDir()
	src := createTestJPEG(t, dir, "test.jpg")

	ts := NewThumbnailService(dir)
	thumbs, err := ts.GenerateAll("test-file-id", src, "image/jpeg")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	if len(thumbs) != 5 {
		t.Fatalf("expected 5 thumbnails, got %d", len(thumbs))
	}

	for _, th := range thumbs {
		info, err := os.Stat(th.LocalPath)
		if err != nil {
			t.Errorf("thumbnail file missing after generation: %s (%v)", th.LocalPath, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("thumbnail file is empty: %s", th.LocalPath)
		}
	}
}

func TestThumbnailService_GenerateVideoProxy_shouldGenerate720pFor1080p(t *testing.T) {
	dir := t.TempDir()
	ts := NewThumbnailService(dir)
	thumbs, err := ts.GenerateAll("test-vid-proxy", "/tmp/test_video_1080p.mp4", "video/mp4")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	var proxy *model.Thumbnail
	for _, th := range thumbs {
		if th.Size == model.ThumbSizeVideoProxy {
			proxy = th
			break
		}
	}
	if proxy == nil {
		t.Fatal("expected video_proxy thumbnail for 1080p video, got nil")
	}
	if proxy.Format != "mp4" {
		t.Errorf("expected format mp4, got %s", proxy.Format)
	}
	if proxy.SizeBytes == 0 {
		t.Error("video proxy file is empty")
	}

	stillsFound := false
	for _, th := range thumbs {
		if th.Size == model.ThumbSizeVideoStill {
			stillsFound = true
			break
		}
	}
	if !stillsFound {
		t.Error("expected video_still thumbnail alongside proxy")
	}
}

func TestThumbnailService_GenerateVideoProxy_shouldSkipFor720pVideo(t *testing.T) {
	dir := t.TempDir()
	ts := NewThumbnailService(dir)
	thumbs, err := ts.GenerateAll("test-vid-noproxy", "/tmp/test_video_720p.mp4", "video/mp4")
	if err != nil {
		t.Fatalf("GenerateAll failed: %v", err)
	}

	for _, th := range thumbs {
		if th.Size == model.ThumbSizeVideoProxy {
			t.Error("should not generate video_proxy for ≤720p video")
		}
	}

	stillsFound := false
	for _, th := range thumbs {
		if th.Size == model.ThumbSizeVideoStill {
			stillsFound = true
			break
		}
	}
	if !stillsFound {
		t.Error("expected video_still thumbnail")
	}
}
