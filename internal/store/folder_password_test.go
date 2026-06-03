package store

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestFolderPasswordStore_Create_shouldSetPassword(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fps := NewFolderPasswordStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Secret Folder", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	expiresAt := time.Now().UTC().Add(30 * time.Minute)

	fp, err := fps.Create(folder.ID, string(hash), "", expiresAt)
	if err != nil {
		t.Fatalf("create folder password: %v", err)
	}
	if fp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if fp.FolderID != folder.ID {
		t.Errorf("expected folder_id %q, got %q", folder.ID, fp.FolderID)
	}
}

func TestFolderPasswordStore_FindByFolderID_shouldReturnPassword(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fps := NewFolderPasswordStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Secret Folder", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	expiresAt := time.Now().UTC().Add(30 * time.Minute)
	fps.Create(folder.ID, string(hash), "", expiresAt)

	found, err := fps.FindByFolderID(folder.ID)
	if err != nil {
		t.Fatalf("find by folder id: %v", err)
	}
	if found.PasswordHash == "" {
		t.Error("expected non-empty password hash")
	}
}

func TestFolderPasswordStore_FindByFolderID_shouldErrorWhenNotFound(t *testing.T) {
	db := OpenTestDB(t)
	fps := NewFolderPasswordStore(db)

	_, err := fps.FindByFolderID("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent folder password")
	}
}

func TestFolderPasswordStore_Delete_shouldRemovePassword(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fps := NewFolderPasswordStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Secret Folder", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	fps.Create(folder.ID, string(hash), "", time.Now().UTC().Add(30*time.Minute))

	if err := fps.DeleteByFolderID(folder.ID); err != nil {
		t.Fatalf("delete by folder id: %v", err)
	}

	_, err := fps.FindByFolderID(folder.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestFolderPasswordStore_DeleteExpired_shouldRemoveExpired(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fps := NewFolderPasswordStore(db)

	user := createTestUser(t, us)
	folder1, _ := fs.Create(user.ID, "F1", nil)
	folder2, _ := fs.Create(user.ID, "F2", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)

	fps.Create(folder1.ID, string(hash), "", time.Now().UTC().Add(-1*time.Hour))
	fps.Create(folder2.ID, string(hash), "", time.Now().UTC().Add(1*time.Hour))

	n, err := fps.DeleteExpired()
	if err != nil {
		t.Fatalf("delete expired: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 deleted, got %d", n)
	}

	_, err = fps.FindByFolderID(folder1.ID)
	if err == nil {
		t.Error("expired folder password should be deleted")
	}
}

func TestFolderPasswordStore_Create_shouldStoreHint(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fps := NewFolderPasswordStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Hint Folder", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	expiresAt := time.Now().UTC().Add(30 * time.Minute)

	fp, err := fps.Create(folder.ID, string(hash), "My birthday", expiresAt)
	if err != nil {
		t.Fatalf("create folder password with hint: %v", err)
	}
	if fp.PasswordHint != "My birthday" {
		t.Errorf("expected hint 'My birthday', got %q", fp.PasswordHint)
	}
}

func TestFolderPasswordStore_FindByFolderID_shouldReturnHint(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fps := NewFolderPasswordStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Hint Folder", nil)
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.MinCost)
	fps.Create(folder.ID, string(hash), "My birthday", time.Now().UTC().Add(30*time.Minute))

	found, err := fps.FindByFolderID(folder.ID)
	if err != nil {
		t.Fatalf("find by folder id: %v", err)
	}
	if found.PasswordHint != "My birthday" {
		t.Errorf("expected hint 'My birthday', got %q", found.PasswordHint)
	}
}
