package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type AlbumShareStore struct {
	db *DB
}

func NewAlbumShareStore(db *DB) *AlbumShareStore {
	return &AlbumShareStore{db: db}
}

func (s *AlbumShareStore) Add(albumID, sharedWithUserID, permission string) (*model.AlbumShare, error) {
	share := &model.AlbumShare{
		ID:               uuid.New().String(),
		AlbumID:          albumID,
		SharedWithUserID: sharedWithUserID,
		Permission:       permission,
		CreatedAt:        time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO album_shares (id, album_id, shared_with_user_id, permission, created_at) VALUES (?, ?, ?, ?, ?)`,
		share.ID, share.AlbumID, share.SharedWithUserID, share.Permission, share.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("add album share: %w", err)
	}

	return share, nil
}

func (s *AlbumShareStore) Remove(id string) error {
	_, err := s.db.Exec(`DELETE FROM album_shares WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("remove album share: %w", err)
	}
	return nil
}

func (s *AlbumShareStore) FindByAlbumAndUser(albumID, userID string) (*model.AlbumShare, error) {
	share := &model.AlbumShare{}
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, album_id, shared_with_user_id, permission, created_at FROM album_shares WHERE album_id = ? AND shared_with_user_id = ?`,
		albumID, userID,
	).Scan(&share.ID, &share.AlbumID, &share.SharedWithUserID, &share.Permission, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("find album share: %w", err)
	}
	share.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return share, nil
}
