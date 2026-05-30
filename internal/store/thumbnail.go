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
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		_, err := s.db.Exec(
			`INSERT OR REPLACE INTO thumbnails (file_id, size, width, height, format, local_path, s3_key, size_bytes, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			t.FileID, t.Size, t.Width, t.Height, t.Format, t.LocalPath, t.S3Key, t.SizeBytes, t.CreatedAt.Format(time.RFC3339),
		)
		if err == nil {
			return nil
		}
		if !isSQLiteBusy(err) {
			return fmt.Errorf("insert thumbnail: %w", err)
		}
		lastErr = err
	}

	return fmt.Errorf("insert thumbnail after %d retries: %w", maxRetries, lastErr)
}

func (s *ThumbnailStore) TotalSize() (int64, error) {
	var size int64
	err := s.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM thumbnails`).Scan(&size)
	return size, err
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

func (s *ThumbnailStore) CountByFileID(fileID string) (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM thumbnails WHERE file_id = ?`, fileID).Scan(&count)
	return count, err
}

type ThumbnailBreakdown struct {
	Size      string `json:"size"`
	Count     int64  `json:"count"`
	TotalSize int64  `json:"total_size"`
}

func (s *ThumbnailStore) Breakdown() ([]ThumbnailBreakdown, error) {
	rows, err := s.db.Query(`SELECT size, COUNT(*) as count, COALESCE(SUM(size_bytes), 0) as total_size FROM thumbnails GROUP BY size ORDER BY size`)
	if err != nil {
		return nil, fmt.Errorf("thumbnail breakdown query: %w", err)
	}
	defer rows.Close()

	var result []ThumbnailBreakdown
	for rows.Next() {
		var b ThumbnailBreakdown
		if err := rows.Scan(&b.Size, &b.Count, &b.TotalSize); err != nil {
			return nil, fmt.Errorf("scan thumbnail breakdown: %w", err)
		}
		result = append(result, b)
	}
	return result, rows.Err()
}
