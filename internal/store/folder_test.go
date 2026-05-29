package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
)

func TestFolderStore_Create_shouldPersistFolder(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	f, err := fs.Create(user.ID, "Test Folder", nil)
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if f.ID == "" {
		t.Error("expected non-empty ID")
	}
	if f.Name != "Test Folder" {
		t.Errorf("expected 'Test Folder', got %q", f.Name)
	}
}

func TestFolderStore_Create_shouldCreateNested(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	parent, _ := fs.Create(user.ID, "Parent", nil)
	child, err := fs.Create(user.ID, "Child", &parent.ID)
	if err != nil {
		t.Fatalf("create nested folder: %v", err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Errorf("expected child.parent_id to be %q, got %v", parent.ID, child.ParentID)
	}
}

func TestFolderStore_ListByUser_shouldReturnUserFolders(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user1 := createTestUser(t, us)
	_, _ = us.Create("folder_user_2", "password123", model.RoleMember, nil)
	user2, _ := us.FindByUsername("folder_user_2")
	if user2 == nil {
		t.Fatal("should find user2")
	}

	fs.Create(user1.ID, "User1 Folder A", nil)
	fs.Create(user1.ID, "User1 Folder B", nil)
	fs.Create(user2.ID, "User2 Folder", nil)

	folders, err := fs.ListByUser(user1.ID)
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(folders) != 2 {
		t.Errorf("expected 2 folders for user1, got %d", len(folders))
	}
}

func TestFolderStore_ListTree_shouldBuildNestedTree(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	parent, _ := fs.Create(user.ID, "Parent", nil)
	fs.Create(user.ID, "Child", &parent.ID)
	fs.Create(user.ID, "Root Only", nil)

	root, err := fs.ListTree(user.ID)
	if err != nil {
		t.Fatalf("list tree: %v", err)
	}

	if len(root.Children) != 2 {
		t.Errorf("expected 2 root-level children, got %d", len(root.Children))
	}

	parentNode := root.Children[0]
	if parentNode.Folder.Name == "Parent" && len(parentNode.Children) != 1 {
		t.Errorf("expected Parent to have 1 child, got %d", len(parentNode.Children))
	}
}

func TestFolderStore_UpdateName_shouldRename(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	f, _ := fs.Create(user.ID, "Old Name", nil)

	err := fs.UpdateName(f.ID, "New Name")
	if err != nil {
		t.Fatalf("update name: %v", err)
	}

	updated, _ := fs.FindByID(f.ID)
	if updated.Name != "New Name" {
		t.Errorf("expected 'New Name', got %q", updated.Name)
	}
}

func TestFolderStore_Delete_shouldRemoveFolder(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	f, _ := fs.Create(user.ID, "To Delete", nil)

	if err := fs.Delete(f.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err := fs.FindByID(f.ID)
	if err == nil {
		t.Error("expected error finding deleted folder")
	}
}
