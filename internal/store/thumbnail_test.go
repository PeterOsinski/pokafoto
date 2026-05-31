package store

import (
	"testing"

	"github.com/drive/drive/internal/model"
)

func TestThumbnailStore_TotalSize_shouldSumAllSizes(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewThumbnailStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "photo1.jpg")
	f2 := createTestFile(t, fs, user.ID, "photo2.jpg")

	createTestThumb(t, ts, f1.ID, model.ThumbSizeSmall, 500)
	createTestThumb(t, ts, f1.ID, model.ThumbSizeMedium, 1200)
	createTestThumb(t, ts, f2.ID, model.ThumbSizeSmall, 800)

	total, err := ts.TotalSize()
	if err != nil {
		t.Fatalf("TotalSize: %v", err)
	}
	if total != 2500 {
		t.Errorf("expected 2500, got %d", total)
	}
}

func TestThumbnailStore_TotalSize_shouldReturnZeroWhenEmpty(t *testing.T) {
	db := OpenTestDB(t)
	ts := NewThumbnailStore(db)

	total, err := ts.TotalSize()
	if err != nil {
		t.Fatalf("TotalSize: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0, got %d", total)
	}
}

func TestThumbnailStore_Create_shouldAcceptXL(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewThumbnailStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "photo.jpg")

	createTestThumb(t, ts, f.ID, model.ThumbSizeXL, 500000)
	createTestThumb(t, ts, f.ID, model.ThumbSizeSmall, 500)
	createTestThumb(t, ts, f.ID, model.ThumbSizeLarge, 1200)
	createTestThumb(t, ts, f.ID, model.ThumbSizeMedium, 2000)

	thumb, err := ts.FindByFileIDAndSize(f.ID, model.ThumbSizeXL)
	if err != nil {
		t.Fatalf("FindByFileIDAndSize xl: %v", err)
	}
	if thumb.SizeBytes != 500000 {
		t.Errorf("expected 500000, got %d", thumb.SizeBytes)
	}
	if string(thumb.Size) != string(model.ThumbSizeXL) {
		t.Errorf("expected xl, got %s", thumb.Size)
	}

	count, err := ts.CountByFileID(f.ID)
	if err != nil {
		t.Fatalf("CountByFileID: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 thumbnails, got %d", count)
	}

	breakdown, err := ts.Breakdown()
	if err != nil {
		t.Fatalf("Breakdown: %v", err)
	}
	hasXL := false
	for _, b := range breakdown {
		if b.Size == "xl" {
			hasXL = true
			break
		}
	}
	if !hasXL {
		t.Error("expected xl size in breakdown")
	}
}

func createTestThumb(t *testing.T, ts *ThumbnailStore, fileID string, size model.ThumbnailSize, sizeBytes int64) {
	t.Helper()
	th := &model.Thumbnail{
		FileID:    fileID,
		Size:      size,
		Width:     200,
		Height:    200,
		Format:    "jpeg",
		LocalPath: "/tmp/thumb-" + string(size) + ".jpg",
		SizeBytes: sizeBytes,
	}
	if err := ts.Create(th); err != nil {
		t.Fatalf("create test thumbnail: %v", err)
	}
}
