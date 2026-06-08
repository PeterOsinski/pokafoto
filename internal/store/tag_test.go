package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

func TestTagStore_FindOrCreate_shouldCreateTag(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	store := NewTagStore(db)
	tag, err := store.FindOrCreate("vacation")
	if err != nil {
		t.Fatalf("FindOrCreate() error = %v", err)
	}
	if tag.Name != "vacation" {
		t.Errorf("expected 'vacation', got %q", tag.Name)
	}

	tag2, err := store.FindOrCreate("vacation")
	if err != nil {
		t.Fatalf("FindOrCreate() error = %v", err)
	}
	if tag2.ID != tag.ID {
		t.Error("expected same tag instance")
	}
}

func TestTagStore_Search_shouldReturnMatchingTags(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	store := NewTagStore(db)
	store.FindOrCreate("beach")
	store.FindOrCreate("birthday")
	store.FindOrCreate("business")

	tags, err := store.Search("b")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(tags) != 3 {
		t.Errorf("expected 3 tags starting with 'b', got %d", len(tags))
	}
}

func TestTagStore_AddToFile_shouldAddTag(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")
	store := NewTagStore(db)
	tag, _ := store.FindOrCreate("vacation")

	if err := store.AddToFile(file.ID, tag.ID, u.ID); err != nil {
		t.Fatalf("AddToFile() error = %v", err)
	}

	tags, err := store.FindByFileID(file.ID)
	if err != nil {
		t.Fatalf("FindByFileID() error = %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "vacation" {
		t.Error("expected file to have vacation tag")
	}
}

func TestTagStore_ListWithCount_shouldReturnCounts(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewTagStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "beach.jpg")
	f2 := createTestFile(t, fs, user.ID, "sunset.jpg")

	tag1, _ := ts.FindOrCreate("vacation")
	tag2, _ := ts.FindOrCreate("landscape")
	ts.AddToFile(f1.ID, tag1.ID, user.ID)
	ts.AddToFile(f2.ID, tag1.ID, user.ID)
	ts.AddToFile(f2.ID, tag2.ID, user.ID)

	tags, err := ts.ListWithCount(user.ID)
	if err != nil {
		t.Fatalf("ListWithCount() error = %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tag results, got %d", len(tags))
	}
	if tags[0].Name != "vacation" || tags[0].Count != 2 {
		t.Errorf("expected vacation with count 2, got %s=%d", tags[0].Name, tags[0].Count)
	}
	if tags[1].Name != "landscape" || tags[1].Count != 1 {
		t.Errorf("expected landscape with count 1, got %s=%d", tags[1].Name, tags[1].Count)
	}
}

func TestTagStore_RemoveFromFile_shouldRemoveTag(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewTagStore(db)

	user := createTestUser(t, us)
	file := createTestFile(t, fs, user.ID, "test.jpg")
	tag, _ := ts.FindOrCreate("vacation")
	ts.AddToFile(file.ID, tag.ID, user.ID)

	if err := ts.RemoveFromFile(file.ID, tag.ID); err != nil {
		t.Fatalf("RemoveFromFile() error = %v", err)
	}

	tags, _ := ts.FindByFileID(file.ID)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after removal, got %d", len(tags))
	}
}

func TestTagStore_ListWithCount_shouldExcludeDeletedAndOtherUsers(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewTagStore(db)

	user1 := createTestUser(t, us)
	user2, err := us.Create("tagcountuser2_"+uuid.NewString()[:8], "password123", model.RoleMember, nil)
	if err != nil {
		t.Fatalf("create user2: %v", err)
	}
	f1 := createTestFile(t, fs, user1.ID, "beach.jpg")
	f2 := createTestFile(t, fs, user2.ID, "other.jpg")

	tag1, _ := ts.FindOrCreate("vacation")
	ts.AddToFile(f1.ID, tag1.ID, user1.ID)
	ts.AddToFile(f2.ID, tag1.ID, user2.ID)

	tags, err := ts.ListWithCount(user1.ID)
	if err != nil {
		t.Fatalf("ListWithCount() error = %v", err)
	}
	if len(tags) != 1 || tags[0].Count != 1 {
		t.Errorf("expected only user1's file to count, got %d tags", len(tags))
	}

	fs.SoftDelete(f1.ID)
	tags, err = ts.ListWithCount(user1.ID)
	if err != nil {
		t.Fatalf("ListWithCount() after soft delete error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("expected 0 tags after soft delete, got %d", len(tags))
	}
}
