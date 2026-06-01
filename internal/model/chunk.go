package model

import "time"

type ChunkStatus string

const (
	ChunkStatusPending ChunkStatus = "pending"
	ChunkStatusStored  ChunkStatus = "stored"
)

type ChunkRecord struct {
	UploadID    string      `json:"upload_id" db:"upload_id"`
	ChunkIndex  int         `json:"chunk_index" db:"chunk_index"`
	ChunkSize   int64       `json:"chunk_size" db:"chunk_size"`
	Offset      int64       `json:"offset" db:"offset"`
	Status      ChunkStatus `json:"status" db:"status"`
	ChunkSHA256 *string     `json:"chunk_sha256,omitempty" db:"chunk_sha256"`
	TempPath    *string     `json:"temp_path,omitempty" db:"temp_path"`
	CreatedAt   time.Time   `json:"created_at" db:"created_at"`
}
