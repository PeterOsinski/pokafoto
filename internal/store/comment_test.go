package store

import (
	"testing"
)

func TestCommentStore_Create_shouldCreateComment(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")

	commentStore := NewCommentStore(db)
	c, err := commentStore.Create(file.ID, u.ID, "Great photo!")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if c.Content != "Great photo!" {
		t.Errorf("expected 'Great photo!', got %q", c.Content)
	}
	if c.FileID != file.ID {
		t.Errorf("expected fileID %q, got %q", file.ID, c.FileID)
	}
}

func TestCommentStore_FindByFileID_shouldReturnComments(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")

	commentStore := NewCommentStore(db)
	commentStore.Create(file.ID, u.ID, "First")
	commentStore.Create(file.ID, u.ID, "Second")

	comments, err := commentStore.FindByFileID(file.ID)
	if err != nil {
		t.Fatalf("FindByFileID() error = %v", err)
	}
	if len(comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(comments))
	}
}

func TestCommentStore_Delete_shouldDeleteOwnComment(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")

	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Delete me")

	if err := commentStore.Delete(c.ID, u.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := commentStore.FindByID(c.ID)
	if err == nil {
		t.Error("expected comment to be deleted")
	}
}
