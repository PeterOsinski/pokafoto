package store

import (
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestFolderShareStore_Create_shouldPersistShare(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)

	share := &model.FolderShare{
		FolderID:    folder.ID,
		Permissions: model.ShareRead,
	}
	if err := shs.Create(share); err != nil {
		t.Fatalf("create share: %v", err)
	}
	if share.ID == "" {
		t.Error("expected non-empty ID")
	}
	if share.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestFolderShareStore_Create_withPassword_shouldStoreHash(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)

	hash, _ := bcrypt.GenerateFromPassword([]byte("sharepass"), bcrypt.MinCost)
	hashStr := string(hash)
	share := &model.FolderShare{
		FolderID:     folder.ID,
		Permissions:  model.ShareReadUpload,
		HasPassword:  true,
		PasswordHash: &hashStr,
	}
	if err := shs.Create(share); err != nil {
		t.Fatalf("create share with password: %v", err)
	}

	found, err := shs.FindByToken(share.Token)
	if err != nil {
		t.Fatalf("find by token: %v", err)
	}
	if !found.HasPassword {
		t.Error("expected share to have password")
	}
	if found.PasswordHash == nil || *found.PasswordHash == "" {
		t.Error("expected non-empty password hash")
	}
}

func TestFolderShareStore_Create_withExpiry_shouldStoreExpiry(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)

	expiry := time.Now().UTC().Add(72 * time.Hour)
	share := &model.FolderShare{
		FolderID:    folder.ID,
		Permissions: model.ShareRead,
		ExpiresAt:   &expiry,
	}
	if err := shs.Create(share); err != nil {
		t.Fatalf("create share with expiry: %v", err)
	}

	found, err := shs.FindByToken(share.Token)
	if err != nil {
		t.Fatalf("find by token: %v", err)
	}
	if found.ExpiresAt == nil {
		t.Error("expected non-nil expiry")
	}
}

func TestFolderShareStore_Create_withUploadLimit_shouldPersistLimit(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)

	limit := int64(104857600)
	share := &model.FolderShare{
		FolderID:         folder.ID,
		Permissions:      model.ShareReadUpload,
		UploadLimitBytes: &limit,
	}
	if err := shs.Create(share); err != nil {
		t.Fatalf("create share with limit: %v", err)
	}

	found, err := shs.FindByToken(share.Token)
	if err != nil {
		t.Fatalf("find by token: %v", err)
	}
	if found.UploadLimitBytes == nil || *found.UploadLimitBytes != limit {
		t.Errorf("expected upload limit %d, got %v", limit, found.UploadLimitBytes)
	}
}

func TestFolderShareStore_FindByToken_shouldErrorOnNonExistent(t *testing.T) {
	db := OpenTestDB(t)
	shs := NewFolderShareStore(db)

	_, err := shs.FindByToken("nonexistent-token")
	if err == nil {
		t.Error("expected error for nonexistent share")
	}
}

func TestFolderShareStore_ListByFolder_shouldReturnFolderShares(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder1, _ := fs.Create(user.ID, "F1", nil)
	folder2, _ := fs.Create(user.ID, "F2", nil)

	shs.Create(&model.FolderShare{FolderID: folder1.ID, Permissions: model.ShareRead})
	shs.Create(&model.FolderShare{FolderID: folder1.ID, Permissions: model.ShareReadUpload})
	shs.Create(&model.FolderShare{FolderID: folder2.ID, Permissions: model.ShareRead})

	shares, err := shs.ListByFolder(folder1.ID)
	if err != nil {
		t.Fatalf("list by folder: %v", err)
	}
	if len(shares) != 2 {
		t.Errorf("expected 2 shares, got %d", len(shares))
	}
}

func TestFolderShareStore_Delete_shouldRemoveShare(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)

	share := &model.FolderShare{FolderID: folder.ID, Permissions: model.ShareRead}
	shs.Create(share)

	if err := shs.Delete(share.ID); err != nil {
		t.Fatalf("delete share: %v", err)
	}

	_, err := shs.FindByID(share.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestFolderShareStore_Update_shouldChangePermissions(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	shs := NewFolderShareStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)

	share := &model.FolderShare{FolderID: folder.ID, Permissions: model.ShareRead}
	shs.Create(share)

	var newLimit *int64
	var newExpiry *time.Time
	var newHash *string
	err := shs.Update(share.ID, model.ShareReadWrite, newLimit, newExpiry, false, newHash)
	if err != nil {
		t.Fatalf("update share: %v", err)
	}

	found, err := shs.FindByID(share.ID)
	if err != nil {
		t.Fatalf("find after update: %v", err)
	}
	if found.Permissions != model.ShareReadWrite {
		t.Errorf("expected read_write, got %q", found.Permissions)
	}
}
