package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type AlbumItemStore struct {
	db *DB
}

func NewAlbumItemStore(db *DB) *AlbumItemStore {
	return &AlbumItemStore{db: db}
}

func (s *AlbumItemStore) Add(albumID, fileID, addedByUserID string) (*model.AlbumItem, error) {
	item := &model.AlbumItem{
		ID:            uuid.New().String(),
		AlbumID:       albumID,
		FileID:        fileID,
		AddedByUserID: addedByUserID,
		SortOrder:     0,
		CreatedAt:     time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO album_items (id, album_id, file_id, added_by_user_id, sort_order, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.AlbumID, item.FileID, item.AddedByUserID, item.SortOrder, item.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("add album item: %w", err)
	}

	return item, nil
}

func (s *AlbumItemStore) Remove(albumID, fileID string) error {
	_, err := s.db.Exec(`DELETE FROM album_items WHERE album_id = ? AND file_id = ?`, albumID, fileID)
	if err != nil {
		return fmt.Errorf("remove album item: %w", err)
	}
	return nil
}

func (s *AlbumItemStore) RemoveByID(id string) error {
	_, err := s.db.Exec(`DELETE FROM album_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("remove album item by id: %w", err)
	}
	return nil
}

func (s *AlbumItemStore) ListFileIDs(albumID string, limit, offset int) ([]string, int64, error) {
	var total int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM album_items WHERE album_id = ?`, albumID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count album items: %w", err)
	}

	rows, err := s.db.Query(
		`SELECT file_id FROM album_items WHERE album_id = ? ORDER BY sort_order, created_at LIMIT ? OFFSET ?`,
		albumID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list album items: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, total, rows.Err()
}

func (s *AlbumItemStore) FindByAlbumAndFile(albumID, fileID string) (*model.AlbumItem, error) {
	item := &model.AlbumItem{}
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, album_id, file_id, added_by_user_id, sort_order, created_at FROM album_items WHERE album_id = ? AND file_id = ?`,
		albumID, fileID,
	).Scan(&item.ID, &item.AlbumID, &item.FileID, &item.AddedByUserID, &item.SortOrder, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("find album item: %w", err)
	}
	item.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return item, nil
}

func (s *AlbumItemStore) HasSharedAccess(fileID, userID string) (bool, error) {
	rows, err := s.db.Query(
		`SELECT 1 FROM album_items ai
		 JOIN album_shares sh ON ai.album_id = sh.album_id
		 WHERE ai.file_id = ? AND sh.shared_with_user_id = ? LIMIT 1`,
		fileID, userID,
	)
	if err != nil {
		return false, fmt.Errorf("check shared access: %w", err)
	}
	defer rows.Close()
	return rows.Next(), nil
}

func (s *AlbumItemStore) GetSharedPermission(fileID, userID string) (string, error) {
	var perm string
	err := s.db.QueryRow(
		`SELECT sh.permission FROM album_items ai
		 JOIN album_shares sh ON ai.album_id = sh.album_id
		 WHERE ai.file_id = ? AND sh.shared_with_user_id = ?
		 ORDER BY CASE sh.permission WHEN 'edit' THEN 0 WHEN 'comment' THEN 1 WHEN 'view' THEN 2 END LIMIT 1`,
		fileID, userID,
	).Scan(&perm)
	if err != nil {
		return "", fmt.Errorf("get shared permission: %w", err)
	}
	return perm, nil
}

func (s *AlbumItemStore) ListAlbumsByFile(fileID, userID string) ([]AlbumFileInfo, error) {
	rows, err := s.db.Query(
		`SELECT a.id, a.name, a.user_id, a.user_id = ? as is_owner
		 FROM albums a
		 JOIN album_items ai ON a.id = ai.album_id
		 WHERE ai.file_id = ?
		 AND (a.user_id = ? OR EXISTS (
			SELECT 1 FROM album_shares s
			WHERE s.album_id = a.id AND s.shared_with_user_id = ?
		 ))
		 ORDER BY a.created_at DESC`,
		userID, fileID, userID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list albums by file: %w", err)
	}
	defer rows.Close()

	var albums []AlbumFileInfo
	for rows.Next() {
		var a AlbumFileInfo
		if err := rows.Scan(&a.ID, &a.Name, &a.OwnerID, &a.IsOwner); err != nil {
			continue
		}
		albums = append(albums, a)
	}
	return albums, rows.Err()
}
