package model

import "time"

type ShareUpload struct {
	ID        string    `json:"id" db:"id"`
	ShareID   string    `json:"share_id" db:"share_id"`
	FileID    string    `json:"file_id" db:"file_id"`
	SizeBytes int64     `json:"size_bytes" db:"size_bytes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
