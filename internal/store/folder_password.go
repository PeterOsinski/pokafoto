package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type FolderPasswordStore struct {
	db *DB
}

func NewFolderPasswordStore(db *DB) *FolderPasswordStore {
	return &FolderPasswordStore{db: db}
}

func (s *FolderPasswordStore) Create(folderID, passwordHash string, expiresAt time.Time) (*model.FolderPassword, error) {
	fp := &model.FolderPassword{
		ID:           uuid.New().String(),
		FolderID:     folderID,
		PasswordHash: passwordHash,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT INTO folder_passwords (id, folder_id, password_hash, expires_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		fp.ID, fp.FolderID, fp.PasswordHash, fp.ExpiresAt.Format(time.RFC3339), fp.CreatedAt.Format(time.RFC3339), fp.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert folder password: %w", err)
	}

	return fp, nil
}

func (s *FolderPasswordStore) FindByFolderID(folderID string) (*model.FolderPassword, error) {
	var id, passwordHash, expiresAtStr, createdAtStr, updatedAtStr string
	err := s.db.QueryRow(
		`SELECT id, folder_id, password_hash, expires_at, created_at, updated_at FROM folder_passwords WHERE folder_id = ?`, folderID,
	).Scan(&id, &folderID, &passwordHash, &expiresAtStr, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("find folder password: %w", err)
	}

	fp := &model.FolderPassword{
		ID:           id,
		FolderID:     folderID,
		PasswordHash: passwordHash,
	}
	fp.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	fp.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	fp.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return fp, nil
}

func (s *FolderPasswordStore) DeleteByFolderID(folderID string) error {
	_, err := s.db.Exec(`DELETE FROM folder_passwords WHERE folder_id = ?`, folderID)
	if err != nil {
		return fmt.Errorf("delete folder password: %w", err)
	}
	return nil
}

func (s *FolderPasswordStore) DeleteExpired() (int64, error) {
	result, err := s.db.Exec(
		`DELETE FROM folder_passwords WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired folder passwords: %w", err)
	}
	n, _ := result.RowsAffected()
	return n, nil
}
