package store

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/drive/drive/internal/model"
)

func createTestFile(t *testing.T, s *FileStore, userID, name string) *model.File {
	t.Helper()
	f := &model.File{
		UserID:       userID,
		Filename:     "2024/07/" + name,
		OriginalName: name,
		Path:         "2024/07",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeSHA256(name),
		MediaType:    model.MediaTypePhoto,
	}
	if err := s.Create(f); err != nil {
		t.Fatalf("create test file: %v", err)
	}
	return f
}

func makeSHA256(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:])
}

func TestFileStore_Create_shouldPersistFile(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := &model.File{
		UserID:       user.ID,
		Filename:     "2024/01/photo.jpg",
		OriginalName: "photo.jpg",
		Path:         "2024/01",
		SizeBytes:    4096,
		MimeType:     "image/jpeg",
		SHA256:       makeSHA256("photo1"),
		MediaType:    model.MediaTypePhoto,
	}

	if err := fs.Create(f); err != nil {
		t.Fatalf("create file: %v", err)
	}
	if f.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestFileStore_Create_shouldAllowDuplicateSHA256(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "dup.jpg")
	f2 := &model.File{
		UserID:    user.ID,
		SHA256:    f1.SHA256,
		MimeType:  "image/jpeg",
		MediaType: model.MediaTypePhoto,
	}
	err := fs.Create(f2)
	if err != nil {
		t.Errorf("expected no error on duplicate SHA256 (constraint removed for folder-scoped dedup), got: %v", err)
	}
}

func TestFileStore_FindByID_shouldReturnFile(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	created := createTestFile(t, fs, user.ID, "find.jpg")

	found, err := fs.FindByID(created.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found == nil {
		t.Fatal("expected file, got nil")
	}
	if found.ID != created.ID {
		t.Errorf("expected id %q, got %q", created.ID, found.ID)
	}
}

func TestFileStore_FindByID_shouldReturnNil(t *testing.T) {
	db := OpenTestDB(t)
	fs := NewFileStore(db)

	f, err := fs.FindByID("nonexistent")
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if f != nil {
		t.Error("expected nil")
	}
}

func TestFileStore_FindBySHA256_shouldReturnFile(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	created := createTestFile(t, fs, user.ID, "hash.jpg")

	found, err := fs.FindBySHA256(user.ID, created.SHA256)
	if err != nil {
		t.Fatalf("find by sha256: %v", err)
	}
	if found == nil {
		t.Fatal("expected file")
	}
}

func TestFileStore_FindByNameAndSize_shouldMatch(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	created := createTestFile(t, fs, user.ID, "namesize.jpg")

	found, err := fs.FindByNameAndSize(user.ID, created.OriginalName, created.SizeBytes)
	if err != nil {
		t.Fatalf("find by name and size: %v", err)
	}
	if found == nil {
		t.Fatal("expected file")
	}
}

func TestFileStore_List_shouldReturnFilesForUser(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user1 := createTestUser(t, us)
	user2, _ := us.Create("user_"+t.Name()+"_2", "password123", model.RoleMember, nil)

	createTestFile(t, fs, user1.ID, "a.jpg")
	createTestFile(t, fs, user1.ID, "b.jpg")
	createTestFile(t, fs, user2.ID, "c.jpg")

	files, _, total, err := fs.List(FileListOptions{
		UserID: user1.ID,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 files for user1, got %d", total)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 items, got %d", len(files))
	}
}

func TestFileStore_List_shouldFilterByPath(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	createTestFile(t, fs, user.ID, "path1.jpg")

	f2 := &model.File{
		UserID:       user.ID,
		Filename:     "2024/06/path2.jpg",
		OriginalName: "path2.jpg",
		Path:         "2024/06",
		SizeBytes:    2048,
		MimeType:     "image/jpeg",
		SHA256:       makeSHA256("path2"),
		MediaType:    model.MediaTypePhoto,
	}
	fs.Create(f2)

	_, _, total, err := fs.List(FileListOptions{
		UserID: user.ID,
		Path:   "2024/06",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("list files by path: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 file in path, got %d", total)
	}
}

func TestFileStore_List_shouldFilterByMediaType(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	createTestFile(t, fs, user.ID, "photo.jpg")

	vf := &model.File{
		UserID:       user.ID,
		Filename:     "2024/07/video.mp4",
		OriginalName: "video.mp4",
		Path:         "2024/07",
		SizeBytes:    8192,
		MimeType:     "video/mp4",
		SHA256:       makeSHA256("video1"),
		MediaType:    model.MediaTypeVideo,
	}
	fs.Create(vf)

	_, _, photoTotal, _ := fs.List(FileListOptions{UserID: user.ID, MediaType: "photo", Limit: 10})
	if photoTotal != 1 {
		t.Errorf("expected 1 photo, got %d", photoTotal)
	}

	_, _, videoTotal, _ := fs.List(FileListOptions{UserID: user.ID, MediaType: "video", Limit: 10})
	if videoTotal != 1 {
		t.Errorf("expected 1 video, got %d", videoTotal)
	}
}

func TestFileStore_List_shouldSortByCreatedAt(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "first.jpg")
	_ = createTestFile(t, fs, user.ID, "second.jpg")

	files, _, _, err := fs.List(FileListOptions{
		UserID: user.ID,
		Sort:   "created_at",
		Order:  "asc",
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("list sorted: %v", err)
	}
	if len(files) < 2 {
		t.Fatal("expected at least 2 files")
	}
	if files[0].ID != f1.ID {
		t.Error("expected first file in ascending order")
	}
}

func TestFileStore_SoftDelete_shouldSetDeleted(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "del.jpg")

	if err := fs.SoftDelete(f.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	found, _ := fs.FindByID(f.ID)
	if found != nil {
		if !found.IsDeleted {
			t.Error("expected file to be soft deleted")
		}
	}
}

func TestFileStore_PermanentDelete_shouldRemoveFile(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "perm.jpg")

	if err := fs.PermanentDelete(f.ID); err != nil {
		t.Fatalf("permanent delete: %v", err)
	}

	found, _ := fs.FindByID(f.ID)
	if found != nil {
		t.Error("expected nil after permanent delete")
	}
}

func TestFileStore_Stats_shouldReturnAggregates(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	createTestFile(t, fs, user.ID, "s1.jpg")

	stats, err := fs.Stats(user.ID)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.TotalFiles < 1 {
		t.Errorf("expected at least 1 file, got %d", stats.TotalFiles)
	}
	if stats.TotalPhotos < 1 {
		t.Errorf("expected at least 1 photo, got %d", stats.TotalPhotos)
	}
}

func TestFileStore_ListDirs_shouldBuildTree(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	createTestFile(t, fs, user.ID, "img1.jpg")

	root, err := fs.ListDirs(user.ID, false)
	if err != nil {
		t.Fatalf("list dirs: %v", err)
	}
	if root == nil {
		t.Fatal("expected root")
	}
	if root.Name != "root" {
		t.Errorf("expected root, got %q", root.Name)
	}
	if root.FileCount < 1 {
		t.Errorf("expected at least 1 file, got %d", root.FileCount)
	}
}

func TestFileStore_Search_shouldFindMatching(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	createTestFile(t, fs, user.ID, "searchable_file.jpg")

	result, err := fs.Search(user.ID, "searchable*", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total < 1 {
		t.Error("expected at least 1 result from FTS5 search")
	}
}

func TestFileStore_Search_shouldReturnEmptyOnNoMatch(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)

	result, err := fs.Search(user.ID, "zzzz_nonexistent_query", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 results, got %d", result.Total)
	}
}

func TestFileStore_Timeline_shouldGroupByMonth(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	takenAt := "2024-07-15T14:30:00Z"
	f := &model.File{
		UserID:       user.ID,
		Filename:     "2024/07/tl1.jpg",
		OriginalName: "tl1.jpg",
		Path:         "2024/07",
		SizeBytes:    1024,
		MimeType:     "image/jpeg",
		SHA256:       makeSHA256("timeline1"),
		MediaType:    model.MediaTypePhoto,
		TakenAt:      &takenAt,
	}
	fs.Create(f)

	groups, err := fs.Timeline(user.ID, "month")
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(groups) == 0 {
		t.Error("expected at least one timeline group when files have taken_at")
	}
}

func TestFileStore_FindByNameAndSizeBatch_shouldFindDuplicates(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "batch_dedup.jpg")
	f2 := createTestFile(t, fs, user.ID, "batch_unique.jpg")

	found, err := fs.FindByNameAndSizeBatch(user.ID, []FileRecord{
		{OriginalName: f1.OriginalName, SizeBytes: f1.SizeBytes},
		{OriginalName: f2.OriginalName, SizeBytes: f2.SizeBytes},
		{OriginalName: "nonexistent.jpg", SizeBytes: 999},
	})
	if err != nil {
		t.Fatalf("find by name and size batch: %v", err)
	}
	if len(found) != 2 {
		t.Errorf("expected 2 matches, got %d", len(found))
	}
}

func TestFileStore_FindByNameAndSizeBatch_shouldReturnEmptyWhenNoDuplicates(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	_ = createTestFile(t, fs, user.ID, "existing.jpg")

	found, err := fs.FindByNameAndSizeBatch(user.ID, []FileRecord{
		{OriginalName: "no_match_1.jpg", SizeBytes: 111},
		{OriginalName: "no_match_2.jpg", SizeBytes: 222},
	})
	if err != nil {
		t.Fatalf("find by name and size batch: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("expected 0 matches, got %d", len(found))
	}
}

func TestFileStore_FindByNameAndSizeBatch_shouldReturnNilOnEmptyInput(t *testing.T) {
	db := OpenTestDB(t)
	fs := NewFileStore(db)

	found, err := fs.FindByNameAndSizeBatch("", nil)
	if err != nil {
		t.Fatalf("find by name and size batch: %v", err)
	}
	if found != nil {
		t.Error("expected nil for empty input")
	}
}

func TestFileStore_AdminFileBreakdown_shouldReturnAggregates(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	createTestFile(t, fs, user.ID, "photo.jpg")

	vf := &model.File{
		UserID:       user.ID,
		Filename:     "2024/07/video.mp4",
		OriginalName: "video.mp4",
		Path:         "2024/07",
		SizeBytes:    8192,
		MimeType:     "video/mp4",
		SHA256:       makeSHA256("video_breakdown"),
		MediaType:    model.MediaTypeVideo,
	}
	if err := fs.Create(vf); err != nil {
		t.Fatalf("create video file: %v", err)
	}

	df := &model.File{
		UserID:       user.ID,
		Filename:     "2024/07/doc.pdf",
		OriginalName: "doc.pdf",
		Path:         "2024/07",
		SizeBytes:    2048,
		MimeType:     "application/pdf",
		SHA256:       makeSHA256("pdf_breakdown"),
		MediaType:    model.MediaTypeFile,
	}
	if err := fs.Create(df); err != nil {
		t.Fatalf("create doc file: %v", err)
	}

	b, err := fs.AdminFileBreakdown()
	if err != nil {
		t.Fatalf("admin file breakdown: %v", err)
	}

	if len(b.MediaTypes) != 3 {
		t.Errorf("expected 3 media types, got %d", len(b.MediaTypes))
	}

	photoFound := false
	videoFound := false
	fileFound := false
	for _, mt := range b.MediaTypes {
		switch mt.MediaType {
		case "photo":
			photoFound = true
			if mt.Count < 1 {
				t.Error("expected photo count >= 1")
			}
		case "video":
			videoFound = true
			if mt.Count != 1 {
				t.Errorf("expected 1 video, got %d", mt.Count)
			}
			if mt.SizeBytes != 8192 {
				t.Errorf("expected 8192 video bytes, got %d", mt.SizeBytes)
			}
		case "file":
			fileFound = true
			if mt.Count != 1 {
				t.Errorf("expected 1 file, got %d", mt.Count)
			}
		}
	}
	if !photoFound || !videoFound || !fileFound {
		t.Error("expected all three media types")
	}

	if len(b.Extensions) < 3 {
		t.Errorf("expected at least 3 extensions, got %d", len(b.Extensions))
	}

	extMap := map[string]bool{"jpeg": false, "mp4": false, "pdf": false}
	for _, e := range b.Extensions {
		if _, ok := extMap[e.Extension]; ok {
			extMap[e.Extension] = true
		}
	}
	for ext, found := range extMap {
		if !found {
			t.Errorf("expected extension %s in breakdown", ext)
		}
	}

	expectedTotal := int64(1024 + 8192 + 2048)
	if b.TotalSize != expectedTotal {
		t.Errorf("expected total size %d, got %d", expectedTotal, b.TotalSize)
	}
}

func TestFileStore_FindPhotosMissingThumbnails_shouldFindPhotos(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewThumbnailStore(db)

	user := createTestUser(t, us)
	photo1 := createTestFile(t, fs, user.ID, "photo1.jpg")
	photo2 := createTestFile(t, fs, user.ID, "photo2.jpg")
	photo3 := createTestFile(t, fs, user.ID, "photo3.jpg")

	createTestThumb(t, ts, photo1.ID, model.ThumbSizeSmall, 100)
	createTestThumb(t, ts, photo1.ID, model.ThumbSizeMedium, 100)
	createTestThumb(t, ts, photo1.ID, model.ThumbSizeLarge, 100)
	createTestThumb(t, ts, photo1.ID, model.ThumbSizePreview, 100)

	createTestThumb(t, ts, photo2.ID, model.ThumbSizeSmall, 100)
	createTestThumb(t, ts, photo2.ID, model.ThumbSizeMedium, 100)
	createTestThumb(t, ts, photo2.ID, model.ThumbSizeLarge, 100)

	files, err := fs.FindPhotosMissingThumbnails()
	if err != nil {
		t.Fatalf("FindPhotosMissingThumbnails: %v", err)
	}

	foundPhoto2 := false
	foundPhoto3 := false
	for _, f := range files {
		if f.ID == photo2.ID {
			foundPhoto2 = true
		}
		if f.ID == photo3.ID {
			foundPhoto3 = true
		}
	}
	if foundPhoto1 := containsID(files, photo1.ID); foundPhoto1 {
		t.Error("photo1 has all thumbnails, should not be in results")
	}
	if !foundPhoto2 {
		t.Error("photo2 is missing preview thumbnail, should be in results")
	}
	if !foundPhoto3 {
		t.Error("photo3 has no thumbnails, should be in results")
	}
}

func containsID(files []*model.File, id string) bool {
	for _, f := range files {
		if f.ID == id {
			return true
		}
	}
	return false
}

func TestFileStore_SoftDelete_shouldSetDeletedAt(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "deleted_at_test.jpg")

	if err := fs.SoftDelete(f.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	found, _ := fs.FindByID(f.ID)
	if found == nil {
		t.Fatal("expected file still exists")
	}
	if !found.IsDeleted {
		t.Error("expected is_deleted = true")
	}
	if found.DeletedAt == nil {
		t.Error("expected deleted_at to be set")
	}
}

func TestFileStore_Restore_shouldClearDeletedAt(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "restore_test.jpg")
	fs.SoftDelete(f.ID)

	if err := fs.Restore(f.ID); err != nil {
		t.Fatalf("restore: %v", err)
	}

	found, _ := fs.FindByID(f.ID)
	if found == nil {
		t.Fatal("expected file still exists")
	}
	if found.IsDeleted {
		t.Error("expected is_deleted = false after restore")
	}
	if found.DeletedAt != nil {
		t.Error("expected deleted_at = nil after restore")
	}
}

func TestFileStore_ListTrash_shouldShowOnlyDeleted(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "trashed.jpg")
	f2 := createTestFile(t, fs, user.ID, "active.jpg")
	fs.SoftDelete(f1.ID)

	files, _, total, err := fs.ListTrash(FileListOptions{
		UserID: user.ID,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("list trash: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 in trash, got %d", total)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 item, got %d", len(files))
	}
	if files[0].ID != f1.ID {
		t.Error("expected trashed file")
	}
	_ = f2
}

func TestFileStore_TrashStats_shouldSumCorrectly(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	_ = createTestFile(t, fs, user.ID, "active1.jpg")
	f1 := createTestFile(t, fs, user.ID, "trash1.jpg")
	f2 := createTestFile(t, fs, user.ID, "trash2.jpg")
	fs.SoftDelete(f1.ID)
	fs.SoftDelete(f2.ID)

	stats, err := fs.TrashStats(user.ID)
	if err != nil {
		t.Fatalf("trash stats: %v", err)
	}
	if stats.Count != 2 {
		t.Errorf("expected 2 in trash, got %d", stats.Count)
	}
	if stats.SizeBytes != 2048 {
		t.Errorf("expected 2048 bytes, got %d", stats.SizeBytes)
	}
}

func TestFileStore_BatchRestore_shouldRestoreMultiple(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "batch1.jpg")
	f2 := createTestFile(t, fs, user.ID, "batch2.jpg")
	fs.SoftDelete(f1.ID)
	fs.SoftDelete(f2.ID)

	if err := fs.BatchRestore(user.ID, []string{f1.ID, f2.ID}); err != nil {
		t.Fatalf("batch restore: %v", err)
	}

	r1, _ := fs.FindByID(f1.ID)
	r2, _ := fs.FindByID(f2.ID)
	if r1 == nil || r1.IsDeleted {
		t.Error("f1 should be restored")
	}
	if r2 == nil || r2.IsDeleted {
		t.Error("f2 should be restored")
	}
}

func TestFileStore_BatchPermanentDelete_shouldRemoveFromDB(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "perm1.jpg")
	f2 := createTestFile(t, fs, user.ID, "perm2.jpg")
	fs.SoftDelete(f1.ID)
	fs.SoftDelete(f2.ID)

	if err := fs.BatchPermanentDelete(user.ID, []string{f1.ID, f2.ID}); err != nil {
		t.Fatalf("batch permanent delete: %v", err)
	}

	r1, _ := fs.FindByID(f1.ID)
	r2, _ := fs.FindByID(f2.ID)
	if r1 != nil {
		t.Error("f1 should be deleted")
	}
	if r2 != nil {
		t.Error("f2 should be deleted")
	}
}

func TestFileStore_GetExpiredFiles_shouldReturnExpired(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "expired.jpg")
	f2 := createTestFile(t, fs, user.ID, "fresh.jpg")
	fs.SoftDelete(f1.ID)
	fs.SoftDelete(f2.ID)

	expired, err := fs.GetExpiredFiles("2099-01-01T00:00:00Z", 100)
	if err != nil {
		t.Fatalf("get expired: %v", err)
	}
	if len(expired) != 2 {
		t.Errorf("expected 2 expired files, got %d", len(expired))
	}

	empty, err := fs.GetExpiredFiles("2000-01-01T00:00:00Z", 100)
	if err != nil {
		t.Fatalf("get expired: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected 0 expired files, got %d", len(empty))
	}
}

func TestFileStore_List_shouldNotShowDeleted(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "hidden.jpg")
	fs.SoftDelete(f.ID)

	_, _, total, err := fs.List(FileListOptions{UserID: user.ID, Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 files in gallery after delete, got %d", total)
	}
}

func TestFileStore_FindBySHA256_shouldNotMatchDeleted(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "dedup_hash.jpg")
	fs.SoftDelete(f.ID)

	found, _ := fs.FindBySHA256(user.ID, f.SHA256)
	if found != nil {
		t.Error("expected nil - deleted files should not match dedup")
	}
}

func TestFileStore_FindByNameAndSize_shouldNotMatchDeleted(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)

	user := createTestUser(t, us)
	f := createTestFile(t, fs, user.ID, "dedup_name.jpg")
	fs.SoftDelete(f.ID)

	found, _ := fs.FindByNameAndSize(user.ID, f.OriginalName, f.SizeBytes)
	if found != nil {
		t.Error("expected nil - deleted files should not match dedup")
	}
}

func TestFileStore_SearchEnhanced_shouldFilterByTags(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewTagStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "sunset_beach.jpg")
	f2 := createTestFile(t, fs, user.ID, "portrait.jpg")

	tag1, _ := ts.FindOrCreate("sunset")
	tag2, _ := ts.FindOrCreate("portrait")
	ts.AddToFile(f1.ID, tag1.ID, user.ID)
	ts.AddToFile(f2.ID, tag2.ID, user.ID)

	result, _, err := fs.SearchEnhanced(SearchOptions{UserID: user.ID, Tags: []string{"sunset"}})
	if err != nil {
		t.Fatalf("search by tag: %v", err)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file with 'sunset' tag, got %d", len(result.Files))
	}
	if result.Files[0].ID != f1.ID {
		t.Errorf("expected file %s, got %s", f1.ID, result.Files[0].ID)
	}
}

func TestFileStore_SearchEnhanced_shouldReturnEmptyForNonMatchingTag(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewTagStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "sunset_beach.jpg")
	tag1, _ := ts.FindOrCreate("sunset")
	ts.AddToFile(f1.ID, tag1.ID, user.ID)

	result, _, err := fs.SearchEnhanced(SearchOptions{UserID: user.ID, Tags: []string{"nonexistent"}})
	if err != nil {
		t.Fatalf("search by tag: %v", err)
	}
	if len(result.Files) != 0 {
		t.Errorf("expected 0 files with 'nonexistent' tag, got %d", len(result.Files))
	}
}

func TestFileStore_SearchEnhanced_shouldMatchCaseInsensitiveTags(t *testing.T) {
	db := OpenTestDB(t)
	us := NewUserStore(db)
	fs := NewFileStore(db)
	ts := NewTagStore(db)

	user := createTestUser(t, us)
	f1 := createTestFile(t, fs, user.ID, "sunset_beach.jpg")
	tag1, _ := ts.FindOrCreate("Sunset")
	ts.AddToFile(f1.ID, tag1.ID, user.ID)

	result, _, err := fs.SearchEnhanced(SearchOptions{UserID: user.ID, Tags: []string{"sunset"}})
	if err != nil {
		t.Fatalf("search by tag: %v", err)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 file, got %d", len(result.Files))
	}
}
