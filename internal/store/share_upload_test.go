package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
)

func TestShareUploadStore_Create_shouldRecordUpload(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	filestore := NewFileStore(db)
	shs := NewFolderShareStore(db)
	sus := NewShareUploadStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)
	share := &model.FolderShare{FolderID: folder.ID, Permissions: model.ShareReadUpload}
	shs.Create(share)

	file := &model.File{
		UserID:       user.ID,
		Filename:     "test.txt",
		OriginalName: "test.txt",
		Path:         "",
		SizeBytes:    1024,
		MimeType:     "text/plain",
		SHA256:       "abc123",
		MediaType:    model.MediaTypeFile,
		FolderID:     &folder.ID,
	}
	filestore.Create(file)

	su, err := sus.Create(share.ID, file.ID, 1024)
	if err != nil {
		t.Fatalf("create share upload: %v", err)
	}
	if su.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestShareUploadStore_SumByShareID_shouldReturnTotal(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	filestore := NewFileStore(db)
	shs := NewFolderShareStore(db)
	sus := NewShareUploadStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)
	share := &model.FolderShare{FolderID: folder.ID, Permissions: model.ShareReadUpload}
	shs.Create(share)

	file1 := &model.File{
		UserID:       user.ID,
		Filename:     "f1.txt",
		OriginalName: "f1.txt",
		Path:         "",
		SizeBytes:    500,
		MimeType:     "text/plain",
		SHA256:       "hash1",
		MediaType:    model.MediaTypeFile,
		FolderID:     &folder.ID,
	}
	filestore.Create(file1)
	sus.Create(share.ID, file1.ID, 500)

	file2 := &model.File{
		UserID:       user.ID,
		Filename:     "f2.txt",
		OriginalName: "f2.txt",
		Path:         "",
		SizeBytes:    700,
		MimeType:     "text/plain",
		SHA256:       "hash2",
		MediaType:    model.MediaTypeFile,
		FolderID:     &folder.ID,
	}
	filestore.Create(file2)
	sus.Create(share.ID, file2.ID, 700)

	total, err := sus.SumByShareID(share.ID)
	if err != nil {
		t.Fatalf("sum by share id: %v", err)
	}
	if total != 1200 {
		t.Errorf("expected total 1200, got %d", total)
	}
}

func TestShareUploadStore_SumByShareID_shouldReturnZeroWhenNone(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	sus := NewShareUploadStore(db)

	total, err := sus.SumByShareID("nonexistent")
	if err != nil {
		t.Fatalf("sum by share id: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0, got %d", total)
	}
}

func TestShareUploadStore_ListByShareID_shouldReturnUploads(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFolderStore(db)
	filestore := NewFileStore(db)
	shs := NewFolderShareStore(db)
	sus := NewShareUploadStore(db)

	user := createTestUser(t, us)
	folder, _ := fs.Create(user.ID, "Shared Folder", nil)
	share := &model.FolderShare{FolderID: folder.ID, Permissions: model.ShareReadUpload}
	shs.Create(share)

	file := &model.File{
		UserID:       user.ID,
		Filename:     "a.txt",
		OriginalName: "a.txt",
		Path:         "",
		SizeBytes:    100,
		MimeType:     "text/plain",
		SHA256:       "hash123",
		MediaType:    model.MediaTypeFile,
		FolderID:     &folder.ID,
	}
	filestore.Create(file)
	sus.Create(share.ID, file.ID, 100)

	uploads, err := sus.ListByShareID(share.ID)
	if err != nil {
		t.Fatalf("ListByShareID: %v", err)
	}
	if len(uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(uploads))
	}
	if uploads[0].FileID != file.ID {
		t.Errorf("expected fileID %q, got %q", file.ID, uploads[0].FileID)
	}
}

func TestShareUploadStore_ListByShareID_shouldReturnEmpty(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	sus := NewShareUploadStore(db)

	uploads, err := sus.ListByShareID("nonexistent")
	if err != nil {
		t.Fatalf("ListByShareID: %v", err)
	}
	if len(uploads) != 0 {
		t.Errorf("expected 0, got %d", len(uploads))
	}
}
