package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
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

func TestCommentStore_Update_shouldModifyContent(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")

	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Original")

	if err := commentStore.Update(c.ID, u.ID, "Modified"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	fetched, _ := commentStore.FindByID(c.ID)
	if fetched.Content != "Modified" {
		t.Errorf("expected 'Modified', got %q", fetched.Content)
	}
}

func TestCommentStore_Update_shouldNotUpdateOtherUsersComment(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u1 := createTestUser(t, us)
	u2, _ := us.Create("comment-other", "password123", model.RoleMember, nil)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u1.ID, "test.jpg")

	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u1.ID, "Original")

	commentStore.Update(c.ID, u2.ID, "Hacked")

	fetched, _ := commentStore.FindByID(c.ID)
	if fetched.Content != "Original" {
		t.Errorf("expected content to remain 'Original' when updated by wrong user, got %q", fetched.Content)
	}
}

func TestCommentStore_FindByID_shouldReturnComment(t *testing.T) {
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")

	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Find me")

	fetched, err := commentStore.FindByID(c.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if fetched == nil {
		t.Fatal("expected comment, got nil")
	}
	if fetched.Content != "Find me" {
		t.Errorf("expected 'Find me', got %q", fetched.Content)
	}
}
