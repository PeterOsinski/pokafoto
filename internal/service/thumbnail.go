package service

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
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

func (s *ThumbnailService) GenerateAll(fileID, sourcePath, mimeType string, orientation *int) ([]*model.Thumbnail, error) {
	isVideo := mimeType == "video/mp4" || mimeType == "video/quicktime" || mimeType == "video/x-msvideo" || mimeType == "video/x-matroska"

	if isVideo {
		return s.generateVideoStills(fileID, sourcePath)
	}

	return s.generateImageThumbs(fileID, sourcePath, mimeType, orientation)
}

func (s *ThumbnailService) generateImageThumbs(fileID, sourcePath, mimeType string, orientation *int) ([]*model.Thumbnail, error) {
	img, err := decodeImage(sourcePath, mimeType)
	if err != nil {
		return nil, err
	}

	if orientation != nil && *orientation >= 2 && *orientation <= 8 && !alreadyNormalized(img, *orientation) {
		img = autoOrient(img, *orientation)
	}

	var thumbs []*model.Thumbnail

	sm, err := s.generateSize(fileID, img, model.ThumbSizeSmall, 60, "jpeg")
	if err == nil {
		thumbs = append(thumbs, sm)
	}

	md, err := s.generateSize(fileID, img, model.ThumbSizeMedium, 600, "jpeg")
	if err == nil {
		thumbs = append(thumbs, md)
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
	defer f.Close()

	if err := jpeg.Encode(f, resized, &jpeg.Options{Quality: 75}); err != nil {
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
	defer f.Close()

	if format == "webp" {
		if err := webp.Encode(f, resized, &webp.Options{Lossless: false, Quality: 80}); err != nil {
			return nil, err
		}
	} else {
		if err := jpeg.Encode(f, resized, &jpeg.Options{Quality: 80}); err != nil {
			return nil, err
		}
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

	args := []string{
		"-y",
		"-ss", "5",
		"-i", sourcePath,
		"-vframes", "1",
		"-q:v", "3",
		"-vf", "scale=600:-1",
		stillPath,
	}

	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ffmpeg video still: %w: %s", err, string(out))
	}

	stat, err := os.Stat(stillPath)
	if err != nil {
		return nil, err
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

func autoOrient(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.FlipH(imaging.Rotate90(img))
	case 6:
		return imaging.Rotate90(img)
	case 7:
		return imaging.FlipH(imaging.Rotate270(img))
	case 8:
		return imaging.Rotate270(img)
	default:
		return img
	}
}

func alreadyNormalized(img image.Image, orientation int) bool {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	switch orientation {
	case 5, 6, 7, 8:
		return h > w
	}
	return false
}

func decodeImage(path, mimeType string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch mimeType {
	case "image/jpeg":
		return jpeg.Decode(f)
	case "image/png":
		return png.Decode(f)
	case "image/webp":
		return webpdecode.Decode(f)
	default:
		img, _, err := image.Decode(f)
		return img, err
	}
}
