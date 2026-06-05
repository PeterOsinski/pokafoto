package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
)

func TestExifStore_Create_shouldPersistExif(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	es := NewExifStore(db)

	user := createTestUser(t, us)
	file := createTestFile(t, fs, user.ID, "exif.jpg")

	camMake := "Canon"
	camModel := "EOS R5"
	iso := 400

	e := &model.ExifData{
		FileID:      file.ID,
		CameraMake:  &camMake,
		CameraModel: &camModel,
		ISO:         &iso,
	}

	if err := es.Create(e); err != nil {
		t.Fatalf("create exif: %v", err)
	}
}

func TestExifStore_FindByFileID_shouldReturnExif(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	es := NewExifStore(db)

	user := createTestUser(t, us)
	file := createTestFile(t, fs, user.ID, "exif2.jpg")

	cam := "Nikon"
	e := &model.ExifData{FileID: file.ID, CameraMake: &cam}
	es.Create(e)

	found, err := es.FindByFileID(file.ID)
	if err != nil {
		t.Fatalf("find by file id: %v", err)
	}
	if found == nil {
		t.Fatal("expected exif")
	}
	if found.CameraMake == nil || *found.CameraMake != "Nikon" {
		t.Error("expected Nikon camera make")
	}
}

func TestExifStore_FindByFileID_shouldReturnNil(t *testing.T) {
	db := OpenTestDB(t)
	es := NewExifStore(db)

	e, err := es.FindByFileID("nonexistent")
	if err != nil {
		t.Fatalf("find by file id: %v", err)
	}
	if e != nil {
		t.Error("expected nil")
	}
}

func TestExifStore_Create_multipleFiles(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	es := NewExifStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "exif_a.jpg")
	f2 := createTestFile(t, fs, user.ID, "exif_b.jpg")

	cam := "Sony"
	if err := es.Create(&model.ExifData{FileID: f1.ID, CameraMake: &cam}); err != nil {
		t.Fatalf("create exif for f1: %v", err)
	}
	if err := es.Create(&model.ExifData{FileID: f2.ID, CameraMake: &cam}); err != nil {
		t.Fatalf("create exif for f2: %v", err)
	}

	e1, _ := es.FindByFileID(f1.ID)
	e2, _ := es.FindByFileID(f2.ID)
	if e1 == nil || e2 == nil {
		t.Fatal("expected exif for both files")
	}
	if e1.FileID == e2.FileID {
		t.Error("expected different file IDs")
	}
}
