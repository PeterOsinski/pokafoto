package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
)

func TestShareUploadStore_Create_shouldRecordUpload(t *testing.T) {
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
