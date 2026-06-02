package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type FolderShareStore struct {
	db *DB
}

func NewFolderShareStore(db *DB) *FolderShareStore {
	return &FolderShareStore{db: db}
}

func (s *FolderShareStore) Create(share *model.FolderShare) error {
	share.ID = uuid.New().String()
	share.Token = uuid.New().String()
	share.CreatedAt = time.Now().UTC()
	share.UpdatedAt = time.Now().UTC()

	var expiresAtStr *string
	if share.ExpiresAt != nil {
		v := share.ExpiresAt.Format(time.RFC3339)
		expiresAtStr = &v
	}

	_, err := s.db.Exec(
		`INSERT INTO folder_shares (id, folder_id, token, permissions, upload_limit_bytes, expires_at, has_password, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		share.ID, share.FolderID, share.Token, string(share.Permissions), share.UploadLimitBytes, expiresAtStr, boolToInt(share.HasPassword), share.PasswordHash, share.CreatedAt.Format(time.RFC3339), share.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert folder share: %w", err)
	}

	return nil
}

func (s *FolderShareStore) FindByID(id string) (*model.FolderShare, error) {
	return s.scanShare(s.db.QueryRow(
		`SELECT id, folder_id, token, permissions, upload_limit_bytes, expires_at, has_password, password_hash, created_at, updated_at FROM folder_shares WHERE id = ?`, id,
	))
}

func (s *FolderShareStore) FindByToken(token string) (*model.FolderShare, error) {
	return s.scanShare(s.db.QueryRow(
		`SELECT id, folder_id, token, permissions, upload_limit_bytes, expires_at, has_password, password_hash, created_at, updated_at FROM folder_shares WHERE token = ?`, token,
	))
}

func (s *FolderShareStore) ListByFolder(folderID string) ([]*model.FolderShare, error) {
	rows, err := s.db.Query(
		`SELECT id, folder_id, token, permissions, upload_limit_bytes, expires_at, has_password, password_hash, created_at, updated_at FROM folder_shares WHERE folder_id = ? ORDER BY created_at DESC`, folderID,
	)
	if err != nil {
		return nil, fmt.Errorf("list folder shares: %w", err)
	}
	defer rows.Close()

	var shares []*model.FolderShare
	for rows.Next() {
		share, err := scanShareFromRows(rows)
		if err != nil {
			continue
		}
		shares = append(shares, share)
	}

	return shares, rows.Err()
}

func (s *FolderShareStore) Update(id string, permissions model.SharePermission, uploadLimitBytes *int64, expiresAt *time.Time, hasPassword bool, passwordHash *string) error {
	var expiresAtStr *string
	if expiresAt != nil {
		v := expiresAt.Format(time.RFC3339)
		expiresAtStr = &v
	}
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`UPDATE folder_shares SET permissions = ?, upload_limit_bytes = ?, expires_at = ?, has_password = ?, password_hash = ?, updated_at = ? WHERE id = ?`,
		string(permissions), uploadLimitBytes, expiresAtStr, boolToInt(hasPassword), passwordHash, now, id,
	)
	if err != nil {
		return fmt.Errorf("update folder share: %w", err)
	}
	return nil
}

func (s *FolderShareStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM folder_shares WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete folder share: %w", err)
	}
	return nil
}

func (s *FolderShareStore) scanShare(row interface{ Scan(...interface{}) error }) (*model.FolderShare, error) {
	var id, folderID, token, permissions string
	var uploadLimitBytes *int64
	var expiresAtStr *string
	var hasPassword int
	var passwordHash *string
	var createdAtStr, updatedAtStr string

	err := row.Scan(&id, &folderID, &token, &permissions, &uploadLimitBytes, &expiresAtStr, &hasPassword, &passwordHash, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scan folder share: %w", err)
	}

	share := &model.FolderShare{
		ID:               id,
		FolderID:         folderID,
		Token:            token,
		Permissions:      model.SharePermission(permissions),
		UploadLimitBytes: uploadLimitBytes,
		HasPassword:      hasPassword == 1,
		PasswordHash:     passwordHash,
	}
	share.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	share.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if expiresAtStr != nil {
		t, _ := time.Parse(time.RFC3339, *expiresAtStr)
		share.ExpiresAt = &t
	}

	return share, nil
}

func scanShareFromRows(rows interface{ Scan(...interface{}) error }) (*model.FolderShare, error) {
	var id, folderID, token, permissions string
	var uploadLimitBytes *int64
	var expiresAtStr *string
	var hasPassword int
	var passwordHash *string
	var createdAtStr, updatedAtStr string

	err := rows.Scan(&id, &folderID, &token, &permissions, &uploadLimitBytes, &expiresAtStr, &hasPassword, &passwordHash, &createdAtStr, &updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scan folder share: %w", err)
	}

	share := &model.FolderShare{
		ID:               id,
		FolderID:         folderID,
		Token:            token,
		Permissions:      model.SharePermission(permissions),
		UploadLimitBytes: uploadLimitBytes,
		HasPassword:      hasPassword == 1,
		PasswordHash:     passwordHash,
	}
	share.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	share.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if expiresAtStr != nil {
		t, _ := time.Parse(time.RFC3339, *expiresAtStr)
		share.ExpiresAt = &t
	}

	return share, nil
}
