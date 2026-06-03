package model

import "time"

type ThumbnailSize string

const (
	ThumbSizeSmall      ThumbnailSize = "sm"
	ThumbSizeMedium     ThumbnailSize = "md"
	ThumbSizeLarge      ThumbnailSize = "lg"
	ThumbSizeXL         ThumbnailSize = "xl"
	ThumbSizePreview    ThumbnailSize = "preview"
	ThumbSizeVideoStill ThumbnailSize = "video_still"
	ThumbSizeVideoProxy ThumbnailSize = "video_proxy"
)

type Thumbnail struct {
	FileID    string        `json:"file_id" db:"file_id"`
	Size      ThumbnailSize `json:"size" db:"size"`
	Width     int           `json:"width" db:"width"`
	Height    int           `json:"height" db:"height"`
	Format    string        `json:"format" db:"format"`
	LocalPath string        `json:"local_path" db:"local_path"`
	S3Key     *string       `json:"s3_key,omitempty" db:"s3_key"`
	SizeBytes int64         `json:"size_bytes" db:"size_bytes"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
}
