package service

import (
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/drive/drive/internal/model"
	webpdecode "golang.org/x/image/webp"
)

type ThumbnailService struct {
	thumbnailsDir string
}

func NewThumbnailService(thumbnailsDir string) *ThumbnailService {
	return &ThumbnailService{thumbnailsDir: thumbnailsDir}
}

func (s *ThumbnailService) GenerateAll(fileID, sourcePath, mimeType string) ([]*model.Thumbnail, error) {
	isVideo := mimeType == "video/mp4" || mimeType == "video/quicktime" || mimeType == "video/x-msvideo" || mimeType == "video/x-matroska"

	if isVideo {
		thumbs, err := s.generateVideoStills(fileID, sourcePath)
		if err != nil {
			return thumbs, err
		}

		proxy, err := s.generateVideoProxy(fileID, sourcePath)
		if err != nil {
			return thumbs, nil
		}
		if proxy != nil {
			thumbs = append(thumbs, proxy)
		}
		return thumbs, nil
	}

	return s.generateImageThumbs(fileID, sourcePath, mimeType)
}

func (s *ThumbnailService) generateImageThumbs(fileID, sourcePath, mimeType string) ([]*model.Thumbnail, error) {
	img, err := decodeImage(sourcePath, mimeType)
	if err != nil {
		return nil, err
	}

	var thumbs []*model.Thumbnail

	sm, err := s.generateSize(fileID, img, model.ThumbSizeSmall, 60, "jpeg")
	if err == nil {
		thumbs = append(thumbs, sm)
	}

	lg, err := s.generateSize(fileID, img, model.ThumbSizeLarge, 300, "jpeg")
	if err == nil {
		thumbs = append(thumbs, lg)
	}

	md, err := s.generateSize(fileID, img, model.ThumbSizeMedium, 600, "jpeg")
	if err == nil {
		thumbs = append(thumbs, md)
	}

	xl, err := s.generateSize(fileID, img, model.ThumbSizeXL, 2000, "jpeg")
	if err == nil {
		thumbs = append(thumbs, xl)
	}

	preview, err := s.generateMaxDim(fileID, img, model.ThumbSizePreview, 720, "webp")
	if err == nil {
		thumbs = append(thumbs, preview)
	}

	return thumbs, nil
}

func (s *ThumbnailService) generateSize(fileID string, img image.Image, size model.ThumbnailSize, width int, format string) (*model.Thumbnail, error) {
	dir := filepath.Join(s.thumbnailsDir, fileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	resized := imaging.Fit(img, width, width, imaging.Lanczos)

	ext := ".jpg"
	thumbPath := filepath.Join(dir, string(size)+ext)

	f, err := os.Create(thumbPath)
	if err != nil {
		return nil, err
	}

	if err := jpeg.Encode(f, resized, &jpeg.Options{Quality: 75}); err != nil {
		f.Close()
		return nil, err
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	stat, _ := os.Stat(thumbPath)
	bounds := resized.Bounds()

	return &model.Thumbnail{
		FileID:    fileID,
		Size:      size,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		Format:    format,
		LocalPath: thumbPath,
		SizeBytes: stat.Size(),
	}, nil
}

func (s *ThumbnailService) generateMaxDim(fileID string, img image.Image, size model.ThumbnailSize, maxDim int, format string) (*model.Thumbnail, error) {
	dir := filepath.Join(s.thumbnailsDir, fileID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	resized := imaging.Fit(img, maxDim, maxDim, imaging.Lanczos)

	ext := ".webp"
	if format == "jpeg" {
		ext = ".jpg"
	}
	thumbPath := filepath.Join(dir, string(size)+ext)

	f, err := os.Create(thumbPath)
	if err != nil {
		return nil, err
	}

	if format == "webp" {
		if err := webp.Encode(f, resized, &webp.Options{Lossless: false, Quality: 80}); err != nil {
			f.Close()
			return nil, err
		}
	} else {
		if err := jpeg.Encode(f, resized, &jpeg.Options{Quality: 80}); err != nil {
			f.Close()
			return nil, err
		}
	}

	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	stat, _ := os.Stat(thumbPath)
	bounds := resized.Bounds()

	return &model.Thumbnail{
		FileID:    fileID,
		Size:      size,
		Width:     bounds.Dx(),
		Height:    bounds.Dy(),
		Format:    format,
		LocalPath: thumbPath,
		SizeBytes: stat.Size(),
	}, nil
}

func (s *ThumbnailService) generateVideoStills(fileID, sourcePath string) ([]*model.Thumbnail, error) {
	stillPath := filepath.Join(s.thumbnailsDir, fileID, "video_still.jpg")
	dir := filepath.Dir(stillPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	for _, seek := range []string{"5", "1", "0.1"} {
		args := []string{
			"-y",
			"-strict", "unofficial",
			"-ss", seek,
			"-i", sourcePath,
			"-vframes", "1",
			"-q:v", "3",
			"-vf", "scale=600:-1",
			stillPath,
		}

		cmd := exec.Command("ffmpeg", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		_ = out

		stat, err := os.Stat(stillPath)
		if err != nil {
			continue
		}

		still := &model.Thumbnail{
			FileID:    fileID,
			Size:      model.ThumbSizeVideoStill,
			Width:     600,
			Height:    338,
			Format:    "jpeg",
			LocalPath: stillPath,
			SizeBytes: stat.Size(),
		}

		return []*model.Thumbnail{still}, nil
	}

	return nil, fmt.Errorf("ffmpeg video still: failed to extract frame at any position")
}

func (s *ThumbnailService) generateVideoProxy(fileID, sourcePath string) (*model.Thumbnail, error) {
	width, height, err := probeVideoResolution(sourcePath)
	if err != nil {
		return nil, err
	}

	if width <= 1280 && height <= 720 {
		return nil, nil
	}

	proxyPath := filepath.Join(s.thumbnailsDir, fileID, "video_proxy.mp4")
	dir := filepath.Dir(proxyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	args := []string{
		"-y",
		"-i", sourcePath,
		"-vf", "scale='min(1280,iw)':'min(720,ih)':force_original_aspect_ratio=decrease",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		proxyPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg video proxy: %w: %s", err, string(out))
	}

	stat, err := os.Stat(proxyPath)
	if err != nil {
		return nil, err
	}

	proxy := &model.Thumbnail{
		FileID:    fileID,
		Size:      model.ThumbSizeVideoProxy,
		Width:     720,
		Height:    405,
		Format:    "mp4",
		LocalPath: proxyPath,
		SizeBytes: stat.Size(),
	}

	return proxy, nil
}

func probeVideoResolution(sourcePath string) (int, int, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		sourcePath,
	}

	cmd := exec.Command("ffprobe", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe: %w: %s", err, string(out))
	}

	var probe struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &probe); err != nil {
		return 0, 0, err
	}

	for _, s := range probe.Streams {
		if s.Width > 0 && s.Height > 0 {
			return s.Width, s.Height, nil
		}
	}

	return 0, 0, fmt.Errorf("no video stream found")
}

func decodeImage(path, mimeType string) (image.Image, error) {
	switch mimeType {
	case "image/webp":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return webpdecode.Decode(f)
	default:
		return imaging.Open(path, imaging.AutoOrientation(true))
	}
}
