package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

func createTestDocFile(t *testing.T, fs *FileStore, userID string) *model.File {
	t.Helper()
	f := &model.File{
		UserID:       userID,
		Filename:     "_app_documents/test-" + uuid.NewString()[:8] + ".md",
		OriginalName: "test.md",
		Path:         "_app_documents",
		SizeBytes:    0,
		MimeType:     "text/markdown",
		SHA256:       uuid.NewString(),
		MediaType:    model.MediaTypeFile,
		IsAppManaged: true,
	}
	if err := fs.Create(f); err != nil {
		t.Fatalf("create test file: %v", err)
	}
	return f
}

func TestDocumentStore_Create_shouldPersistDocument(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	f := createTestDocFile(t, fs, u.ID)
	store := NewDocumentStore(db)

	if err := store.Create(f.ID, "# Hello"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	doc, err := store.FindByFileID(f.ID)
	if err != nil {
		t.Fatalf("FindByFileID() error = %v", err)
	}
	if doc.Content != "# Hello" {
		t.Errorf("expected '# Hello', got %q", doc.Content)
	}
}

func TestDocumentStore_FindByFileID_shouldReturnErrorWhenNotFound(t *testing.T) {
	db := OpenTestDB(t)
	store := NewDocumentStore(db)

	doc, err := store.FindByFileID("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent document")
	}
	if doc != nil {
		t.Error("expected nil document")
	}
}

func TestDocumentStore_UpdateContent_shouldChangeContent(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	f := createTestDocFile(t, fs, u.ID)
	store := NewDocumentStore(db)

	store.Create(f.ID, "original")
	if err := store.UpdateContent(f.ID, "updated"); err != nil {
		t.Fatalf("UpdateContent() error = %v", err)
	}

	doc, _ := store.FindByFileID(f.ID)
	if doc.Content != "updated" {
		t.Errorf("expected 'updated', got %q", doc.Content)
	}
}

func TestDocumentStore_Delete_shouldRemoveDocument(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	f := createTestDocFile(t, fs, u.ID)
	store := NewDocumentStore(db)

	store.Create(f.ID, "some content")
	if err := store.Delete(f.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	doc, err := store.FindByFileID(f.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
	if doc != nil {
		t.Error("expected nil after delete")
	}
}

func TestDocumentStore_Create_shouldAllowEmptyContent(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	f := createTestDocFile(t, fs, u.ID)
	store := NewDocumentStore(db)

	if err := store.Create(f.ID, ""); err != nil {
		t.Fatalf("Create() with empty content error = %v", err)
	}

	doc, err := store.FindByFileID(f.ID)
	if err != nil {
		t.Fatalf("FindByFileID() error = %v", err)
	}
	if doc.Content != "" {
		t.Errorf("expected empty content, got %q", doc.Content)
	}
}
