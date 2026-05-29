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
