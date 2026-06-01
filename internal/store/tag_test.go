package store

import (
	"testing"
)

func TestTagStore_FindOrCreate_shouldCreateTag(t *testing.T) {
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
