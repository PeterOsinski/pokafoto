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

func TestThumbnailStore_SetS3Key_shouldPersistKey(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewThumbnailStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "video.mp4")

	th := &model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeVideoProxy,
		Width:     720,
		Height:    405,
		Format:    "mp4",
		LocalPath: "/tmp/thumb-video_proxy.mp4",
		SizeBytes: 4200000,
	}
	if err := ts.Create(th); err != nil {
		t.Fatalf("create thumbnail: %v", err)
	}

	key := "thumbnails/" + f.ID + "/video_proxy.mp4"
	if err := ts.SetS3Key(f.ID, model.ThumbSizeVideoProxy, key); err != nil {
		t.Fatalf("SetS3Key: %v", err)
	}

	found, err := ts.FindByFileIDAndSize(f.ID, model.ThumbSizeVideoProxy)
	if err != nil {
		t.Fatalf("FindByFileIDAndSize: %v", err)
	}
	if found.S3Key == nil {
		t.Fatal("expected S3Key to be set, got nil")
	}
	if *found.S3Key != key {
		t.Errorf("expected S3Key %q, got %q", key, *found.S3Key)
	}

	newKey := "thumbnails/" + f.ID + "/video_proxy_v2.mp4"
	if err := ts.SetS3Key(f.ID, model.ThumbSizeVideoProxy, newKey); err != nil {
		t.Fatalf("SetS3Key overwrite: %v", err)
	}

	found2, err := ts.FindByFileIDAndSize(f.ID, model.ThumbSizeVideoProxy)
	if err != nil {
		t.Fatalf("FindByFileIDAndSize after overwrite: %v", err)
	}
	if found2.S3Key == nil {
		t.Fatal("expected S3Key after overwrite, got nil")
	}
	if *found2.S3Key != newKey {
		t.Errorf("expected overwritten S3Key %q, got %q", newKey, *found2.S3Key)
	}
}

func TestThumbnailStore_SetS3Key_shouldWorkForVideoStill(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewThumbnailStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "video.mp4")

	th := &model.Thumbnail{
		FileID:    f.ID,
		Size:      model.ThumbSizeVideoStill,
		Width:     600,
		Height:    338,
		Format:    "jpeg",
		LocalPath: "/tmp/thumb-video_still.jpg",
		SizeBytes: 80000,
	}
	if err := ts.Create(th); err != nil {
		t.Fatalf("create thumbnail: %v", err)
	}

	key := "thumbnails/" + f.ID + "/video_still.jpeg"
	if err := ts.SetS3Key(f.ID, model.ThumbSizeVideoStill, key); err != nil {
		t.Fatalf("SetS3Key: %v", err)
	}

	found, err := ts.FindByFileIDAndSize(f.ID, model.ThumbSizeVideoStill)
	if err != nil {
		t.Fatalf("FindByFileIDAndSize: %v", err)
	}
	if found.S3Key == nil {
		t.Fatal("expected S3Key to be set, got nil")
	}
	if *found.S3Key != key {
		t.Errorf("expected S3Key %q, got %q", key, *found.S3Key)
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
