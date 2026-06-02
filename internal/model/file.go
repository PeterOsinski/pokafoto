package model

import "time"

type MediaType string

const (
	MediaTypePhoto MediaType = "photo"
	MediaTypeVideo MediaType = "video"
	MediaTypeFile  MediaType = "file"
)

type File struct {
	ID           string     `json:"id" db:"id"`
	UserID       string     `json:"user_id" db:"user_id"`
	Filename     string     `json:"filename" db:"filename"`
	OriginalName string     `json:"original_name" db:"original_name"`
	Path         string     `json:"path" db:"path"`
	SizeBytes    int64      `json:"size_bytes" db:"size_bytes"`
	MimeType     string     `json:"mime_type" db:"mime_type"`
	SHA256       string     `json:"sha256" db:"sha256"`
	MediaType    MediaType  `json:"media_type" db:"media_type"`
	Width        *int       `json:"width,omitempty" db:"width"`
	Height       *int       `json:"height,omitempty" db:"height"`
	DurationSec  *float64   `json:"duration_sec,omitempty" db:"duration_sec"`
	TakenAt      *string    `json:"taken_at,omitempty" db:"taken_at"`
	FolderID     *string    `json:"folder_id,omitempty" db:"folder_id"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	IsDeleted    bool       `json:"is_deleted" db:"is_deleted"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	IsAppManaged bool       `json:"is_app_managed" db:"is_app_managed"`
}
