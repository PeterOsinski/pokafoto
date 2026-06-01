package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type AlbumStore struct {
	db *DB
}

func NewAlbumStore(db *DB) *AlbumStore {
	return &AlbumStore{db: db}
}

func (s *AlbumStore) Create(userID, name string, description *string) (*model.Album, error) {
	a := &model.Album{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT INTO albums (id, user_id, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, a.UserID, a.Name, a.Description, a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert album: %w", err)
	}

	return a, nil
}

func (s *AlbumStore) FindByID(id string) (*model.Album, error) {
	a := &model.Album{}
	var description *string
	var createdAt, updatedAt string

	err := s.db.QueryRow(
		`SELECT id, user_id, name, description, created_at, updated_at FROM albums WHERE id = ?`, id,
	).Scan(&a.ID, &a.UserID, &a.Name, &description, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("find album: %w", err)
	}

	a.Description = description
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return a, nil
}

func (s *AlbumStore) FindByIDWithOwner(id string) (*model.AlbumWithDetails, error) {
	awd := &model.AlbumWithDetails{Album: &model.Album{}}
	var description *string
	var createdAt, updatedAt string

	err := s.db.QueryRow(
		`SELECT a.id, a.user_id, a.name, a.description, a.created_at, a.updated_at, u.username
		 FROM albums a JOIN users u ON a.user_id = u.id WHERE a.id = ?`, id,
	).Scan(&awd.Album.ID, &awd.Album.UserID, &awd.Album.Name, &description, &createdAt, &updatedAt, &awd.OwnerName)
	if err != nil {
		return nil, fmt.Errorf("find album with owner: %w", err)
	}

	awd.Album.Description = description
	awd.Album.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	awd.Album.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	itemCount, err := s.itemCount(id)
	if err == nil {
		awd.ItemCount = itemCount
	}

	coverFileID, err := s.coverFileID(id)
	if err == nil {
		awd.CoverFileID = coverFileID
	}

	return awd, nil
}

func (s *AlbumStore) itemCount(albumID string) (int64, error) {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM album_items WHERE album_id = ?`, albumID).Scan(&count)
	return count, err
}

func (s *AlbumStore) coverFileID(albumID string) (*string, error) {
	var fileID string
	err := s.db.QueryRow(
		`SELECT file_id FROM album_items WHERE album_id = ? ORDER BY sort_order, created_at LIMIT 1`,
		albumID,
	).Scan(&fileID)
	if err != nil {
		return nil, err
	}
	return &fileID, nil
}

func (s *AlbumStore) ListByUser(userID string) ([]*model.Album, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, description, created_at, updated_at FROM albums WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list albums: %w", err)
	}
	defer rows.Close()

	return scanAlbums(rows)
}

func (s *AlbumStore) ListSharedWithUser(userID string) ([]*model.AlbumWithDetails, error) {
	rows, err := s.db.Query(
		`SELECT a.id, a.user_id, a.name, a.description, a.created_at, a.updated_at, u.username
		 FROM albums a
		 JOIN users u ON a.user_id = u.id
		 JOIN album_shares s ON a.id = s.album_id
		 WHERE s.shared_with_user_id = ?
		 ORDER BY s.created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list shared albums: %w", err)
	}
	defer rows.Close()

	var albums []*model.AlbumWithDetails
	for rows.Next() {
		awd := &model.AlbumWithDetails{Album: &model.Album{}}
		var description *string
		var createdAt, updatedAt string
		if err := rows.Scan(&awd.Album.ID, &awd.Album.UserID, &awd.Album.Name, &description, &createdAt, &updatedAt, &awd.OwnerName); err != nil {
			continue
		}
		awd.Album.Description = description
		awd.Album.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		awd.Album.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		cnt, _ := s.itemCount(awd.Album.ID)
		awd.ItemCount = cnt
		albums = append(albums, awd)
	}

	return albums, rows.Err()
}

func (s *AlbumStore) Update(id, name string, description *string) error {
	_, err := s.db.Exec(
		`UPDATE albums SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("update album: %w", err)
	}
	return nil
}

func (s *AlbumStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM albums WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete album: %w", err)
	}
	return nil
}

func (s *AlbumStore) CheckAccess(albumID, userID string) (permission string, found bool, err error) {
	var ownerID string
	err = s.db.QueryRow(`SELECT user_id FROM albums WHERE id = ?`, albumID).Scan(&ownerID)
	if err != nil {
		return "", false, nil
	}

	if ownerID == userID {
		return "edit", true, nil
	}

	var perm string
	err = s.db.QueryRow(
		`SELECT permission FROM album_shares WHERE album_id = ? AND shared_with_user_id = ?`,
		albumID, userID,
	).Scan(&perm)
	if err != nil {
		return "", false, nil
	}

	return perm, true, nil
}

func scanAlbums(rows *sql.Rows) ([]*model.Album, error) {
	var albums []*model.Album
	for rows.Next() {
		a := &model.Album{}
		var description *string
		var createdAt, updatedAt string
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &description, &createdAt, &updatedAt); err != nil {
			continue
		}
		a.Description = description
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		a.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		albums = append(albums, a)
	}
	return albums, rows.Err()
}

func (s *AlbumStore) ListShares(albumID string) ([]model.SharedUser, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.shared_with_user_id, u.username, s.permission
		 FROM album_shares s JOIN users u ON s.shared_with_user_id = u.id
		 WHERE s.album_id = ?`, albumID,
	)
	if err != nil {
		return nil, fmt.Errorf("list shares: %w", err)
	}
	defer rows.Close()

	var shares []model.SharedUser
	for rows.Next() {
		su := model.SharedUser{}
		if err := rows.Scan(&su.ShareID, &su.UserID, &su.Username, &su.Permission); err != nil {
			continue
		}
		shares = append(shares, su)
	}
	return shares, rows.Err()
}

func (s *AlbumStore) ItemCount(albumID string) int64 {
	count, _ := s.itemCount(albumID)
	return count
}

func (s *AlbumStore) HasShares(albumID string) bool {
	var count int64
	err := s.db.QueryRow(`SELECT COUNT(*) FROM album_shares WHERE album_id = ?`, albumID).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}
