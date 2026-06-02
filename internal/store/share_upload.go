package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type ShareUploadStore struct {
	db *DB
}

func NewShareUploadStore(db *DB) *ShareUploadStore {
	return &ShareUploadStore{db: db}
}

func (s *ShareUploadStore) Create(shareID, fileID string, sizeBytes int64) (*model.ShareUpload, error) {
	su := &model.ShareUpload{
		ID:        uuid.New().String(),
		ShareID:   shareID,
		FileID:    fileID,
		SizeBytes: sizeBytes,
		CreatedAt: time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT INTO share_uploads (id, share_id, file_id, size_bytes, created_at) VALUES (?, ?, ?, ?, ?)`,
		su.ID, su.ShareID, su.FileID, su.SizeBytes, su.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert share upload: %w", err)
	}

	return su, nil
}

func (s *ShareUploadStore) SumByShareID(shareID string) (int64, error) {
	var total *int64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(size_bytes), 0) FROM share_uploads WHERE share_id = ?`,
		shareID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum share uploads: %w", err)
	}
	if total == nil {
		return 0, nil
	}
	return *total, nil
}

func (s *ShareUploadStore) ListByShareID(shareID string) ([]*model.ShareUpload, error) {
	rows, err := s.db.Query(
		`SELECT id, share_id, file_id, size_bytes, created_at FROM share_uploads WHERE share_id = ? ORDER BY created_at DESC`, shareID,
	)
	if err != nil {
		return nil, fmt.Errorf("list share uploads: %w", err)
	}
	defer rows.Close()

	var uploads []*model.ShareUpload
	for rows.Next() {
		var id, shareID, fileID string
		var sizeBytes int64
		var createdAtStr string
		if err := rows.Scan(&id, &shareID, &fileID, &sizeBytes, &createdAtStr); err != nil {
			continue
		}
		createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
		uploads = append(uploads, &model.ShareUpload{
			ID:        id,
			ShareID:   shareID,
			FileID:    fileID,
			SizeBytes: sizeBytes,
			CreatedAt: createdAt,
		})
	}

	return uploads, rows.Err()
}
