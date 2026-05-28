package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
)

type ThumbnailStore struct {
	db *DB
}

func NewThumbnailStore(db *DB) *ThumbnailStore {
	return &ThumbnailStore{db: db}
}

func (s *ThumbnailStore) Create(t *model.Thumbnail) error {
	t.CreatedAt = time.Now().UTC()
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO thumbnails (file_id, size, width, height, format, local_path, s3_key, size_bytes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.FileID, t.Size, t.Width, t.Height, t.Format, t.LocalPath, t.S3Key, t.SizeBytes, t.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert thumbnail: %w", err)
	}
	return nil
}

func (s *ThumbnailStore) FindByFileIDAndSize(fileID string, size model.ThumbnailSize) (*model.Thumbnail, error) {
	t := &model.Thumbnail{}
	var s3Key *string
	var createdAt string

	err := s.db.QueryRow(
		`SELECT file_id, size, width, height, format, local_path, s3_key, size_bytes, created_at FROM thumbnails WHERE file_id = ? AND size = ?`,
		fileID, size,
	).Scan(&t.FileID, &t.Size, &t.Width, &t.Height, &t.Format, &t.LocalPath, &s3Key, &t.SizeBytes, &createdAt)

	if err != nil {
		return nil, err
	}

	t.S3Key = s3Key
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return t, nil
}
