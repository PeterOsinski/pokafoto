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
	thumbs, err := ts.GenerateAll("test-file-id", src, "image/jpeg", nil)
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
	thumbs, err := ts.GenerateAll("test-file-id", src, "image/jpeg", nil)
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

func createOrientedTestJPEG(t *testing.T, dir string, name string, width, height int) string {
	t.Helper()
	path := filepath.Join(dir, name)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	img.Set(50, 30, color.RGBA{255, 0, 0, 255})
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create oriented test jpeg: %v", err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatalf("encode oriented test jpeg: %v", err)
	}
	return path
}

func TestThumbnailService_autoOrient_shouldTransformPixelCoords(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for x := 0; x < 200; x++ {
		for y := 0; y < 100; y++ {
			img.Set(x, y, color.RGBA{128, 128, 128, 255})
		}
	}
	img.Set(50, 30, color.RGBA{255, 0, 0, 255})

	tests := []struct {
		orientation int
		expectX     int
		expectY     int
		expectW     int
		expectH     int
	}{
		{1, 50, 30, 200, 100},
		{2, 149, 30, 200, 100},
		{3, 149, 69, 200, 100},
		{4, 50, 69, 200, 100},
		{5, 69, 149, 100, 200},
		{6, 30, 149, 100, 200},
		{7, 30, 50, 100, 200},
		{8, 69, 50, 100, 200},
	}

	for _, tt := range tests {
		t.Run(formatOrientation(tt.orientation), func(t *testing.T) {
			result := autoOrient(img, tt.orientation)
			bounds := result.Bounds()

			if bounds.Dx() != tt.expectW || bounds.Dy() != tt.expectH {
				t.Errorf("dimensions: expected %dx%d, got %dx%d", tt.expectW, tt.expectH, bounds.Dx(), bounds.Dy())
			}

			r, g, b, _ := result.At(tt.expectX, tt.expectY).RGBA()
			if r != 0xFFFF || g != 0 || b != 0 {
				t.Errorf("pixel at (%d,%d): expected red, got rgba(%d,%d,%d)", tt.expectX, tt.expectY, r>>8, g>>8, b>>8)
			}
		})
	}
}

func TestThumbnailService_GenerateAll_shouldSwapDimensionsOnRotated(t *testing.T) {
	dir := t.TempDir()
	src := createOrientedTestJPEG(t, dir, "rotated.jpg", 800, 600)

	orientations := []struct {
		ori      int
		swapped  bool
	}{
		{1, false},
		{2, false},
		{3, false},
		{4, false},
		{5, true},
		{6, true},
		{7, true},
		{8, true},
	}

	for _, o := range orientations {
		ori := o.ori
		t.Run(formatOrientation(ori), func(t *testing.T) {
			ts := NewThumbnailService(dir)
			thumbs, err := ts.GenerateAll("file-"+formatOrientation(ori), src, "image/jpeg", &ori)
			if err != nil {
				t.Fatalf("GenerateAll: %v", err)
			}

			for _, th := range thumbs {
				if th.Size != model.ThumbSizePreview {
					continue
				}
				if o.swapped {
					if th.Width > th.Height {
						t.Errorf("expected width <= height for orientation %d, got %dx%d", ori, th.Width, th.Height)
					}
				} else {
					if th.Width < th.Height {
						t.Errorf("expected width >= height for orientation %d, got %dx%d", ori, th.Width, th.Height)
					}
				}
			}
		})
	}
}

func TestThumbnailService_GenerateAll_shouldSkipOrientationWhenAlreadyNormalized(t *testing.T) {
	dir := t.TempDir()
	src := createOrientedTestJPEG(t, dir, "portrait.jpg", 600, 800)

	ts := NewThumbnailService(dir)
	ori := 6
	thumbs, err := ts.GenerateAll("portrait-ori6", src, "image/jpeg", &ori)
	if err != nil {
		t.Fatalf("GenerateAll: %v", err)
	}

	for _, th := range thumbs {
		if th.Size != model.ThumbSizePreview {
			continue
		}
		if th.Width > th.Height {
			t.Errorf("portrait image with orientation=6 should NOT be rotated (already normalized), got %dx%d", th.Width, th.Height)
		}
	}
}

func formatOrientation(ori int) string {
	switch ori {
	case 1:
		return "ori1_none"
	case 2:
		return "ori2_flipH"
	case 3:
		return "ori3_rotate180"
	case 4:
		return "ori4_flipV"
	case 5:
		return "ori5_transpose"
	case 6:
		return "ori6_rotate90"
	case 7:
		return "ori7_transverse"
	case 8:
		return "ori8_rotate270"
	default:
		return "ori_unknown"
	}
}
