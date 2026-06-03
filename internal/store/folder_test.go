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

func TestFolderStore_UpdateParent_shouldMoveToNewParent(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	parent1, _ := fs.Create(user.ID, "Parent1", nil)
	parent2, _ := fs.Create(user.ID, "Parent2", nil)
	child, _ := fs.Create(user.ID, "Child", &parent1.ID)

	err := fs.UpdateParent(child.ID, &parent2.ID)
	if err != nil {
		t.Fatalf("update parent: %v", err)
	}

	updated, _ := fs.FindByID(child.ID)
	if updated.ParentID == nil || *updated.ParentID != parent2.ID {
		t.Errorf("expected parent_id %q, got %v", parent2.ID, updated.ParentID)
	}
}

func TestFolderStore_UpdateParent_shouldMoveToRoot(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	parent, _ := fs.Create(user.ID, "Parent", nil)
	child, _ := fs.Create(user.ID, "Child", &parent.ID)

	err := fs.UpdateParent(child.ID, nil)
	if err != nil {
		t.Fatalf("update parent to null: %v", err)
	}

	updated, _ := fs.FindByID(child.ID)
	if updated.ParentID != nil {
		t.Errorf("expected nil parent_id, got %v", updated.ParentID)
	}
}

func TestFolderStore_IsDescendant_shouldDetectDirectChild(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	parent, _ := fs.Create(user.ID, "Parent", nil)
	child, _ := fs.Create(user.ID, "Child", &parent.ID)

	isDesc, err := fs.IsDescendant(child.ID, parent.ID)
	if err != nil {
		t.Fatalf("is descendant: %v", err)
	}
	if !isDesc {
		t.Error("expected child to be descendant of parent")
	}
}

func TestFolderStore_IsDescendant_shouldDetectDeepNesting(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	grandparent, _ := fs.Create(user.ID, "Grand", nil)
	parent, _ := fs.Create(user.ID, "Parent", &grandparent.ID)
	child, _ := fs.Create(user.ID, "Child", &parent.ID)

	isDesc, _ := fs.IsDescendant(child.ID, grandparent.ID)
	if !isDesc {
		t.Error("expected deep descendant to be detected")
	}
}

func TestFolderStore_IsDescendant_shouldRejectUnrelated(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	a, _ := fs.Create(user.ID, "FolderA", nil)
	b, _ := fs.Create(user.ID, "FolderB", nil)

	isDesc, _ := fs.IsDescendant(a.ID, b.ID)
	if isDesc {
		t.Error("unrelated folders should not be descendants")
	}
}

func TestFolderStore_IsDescendant_shouldRejectSelfLoop(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Folder", nil)

	isDesc, _ := fs.IsDescendant(folder.ID, folder.ID)
	if isDesc {
		t.Error("folder should not be descendant of itself")
	}
}

func TestFolderStore_FindDescendantIDs_shouldReturnAllLevels(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	root, _ := fs.Create(user.ID, "Root", nil)
	child1, _ := fs.Create(user.ID, "Child1", &root.ID)
	child2, _ := fs.Create(user.ID, "Child2", &root.ID)
	grandchild, _ := fs.Create(user.ID, "Grandchild", &child1.ID)

	ids, err := fs.FindDescendantIDs(root.ID)
	if err != nil {
		t.Fatalf("find descendants: %v", err)
	}

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}
	if !idSet[root.ID] {
		t.Error("root should be in descendants")
	}
	if !idSet[child1.ID] {
		t.Error("child1 should be in descendants")
	}
	if !idSet[child2.ID] {
		t.Error("child2 should be in descendants")
	}
	if !idSet[grandchild.ID] {
		t.Error("grandchild should be in descendants")
	}
	if len(ids) != 4 {
		t.Errorf("expected 4 descendants, got %d", len(ids))
	}
}

func TestFolderStore_DeleteRecursive_shouldSoftDeleteAllFiles(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	fileStore := NewFileStore(db)

	user := createTestUser(t, us)
	parent, _ := fs.Create(user.ID, "Parent", nil)
	child, _ := fs.Create(user.ID, "Child", &parent.ID)

	f1 := createTestFile(t, fileStore, user.ID, "f1.jpg")
	fileStore.BatchMove(user.ID, []string{f1.ID}, &parent.ID)
	f2 := createTestFile(t, fileStore, user.ID, "f2.jpg")
	fileStore.BatchMove(user.ID, []string{f2.ID}, &child.ID)
	f3 := createTestFile(t, fileStore, user.ID, "f3.jpg")

	result, err := fs.DeleteRecursive(parent.ID, user.ID)
	if err != nil {
		t.Fatalf("delete recursive: %v", err)
	}

	if result.DeletedFiles != 2 {
		t.Errorf("expected 2 deleted files, got %d", result.DeletedFiles)
	}
	if result.DeletedFolders != 2 {
		t.Errorf("expected 2 deleted folders, got %d", result.DeletedFolders)
	}

	_, err = fs.FindByID(parent.ID)
	if err == nil {
		t.Error("parent folder should be deleted")
	}
	_, err = fs.FindByID(child.ID)
	if err == nil {
		t.Error("child folder should be deleted")
	}

	f1Check, _ := fileStore.FindByID(f1.ID)
	if !f1Check.IsDeleted {
		t.Error("f1 should be soft-deleted")
	}
	f2Check, _ := fileStore.FindByID(f2.ID)
	if !f2Check.IsDeleted {
		t.Error("f2 should be soft-deleted")
	}
	f3Check, _ := fileStore.FindByID(f3.ID)
	if f3Check.IsDeleted {
		t.Error("f3 should NOT be soft-deleted")
	}
}

func TestFolderStore_DeleteRecursive_shouldReturnCounts(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Empty", nil)

	result, err := fs.DeleteRecursive(folder.ID, user.ID)
	if err != nil {
		t.Fatalf("delete recursive: %v", err)
	}
	if result.DeletedFiles != 0 {
		t.Errorf("expected 0 deleted files, got %d", result.DeletedFiles)
	}
	if result.DeletedFolders != 1 {
		t.Errorf("expected 1 deleted folder, got %d", result.DeletedFolders)
	}
}
