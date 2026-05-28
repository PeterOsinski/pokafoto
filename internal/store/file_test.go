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

func TestFileStore_Create_shouldReturnErrorOnDuplicateSHA256(t *testing.T) {
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
	if err == nil {
		t.Error("expected error on duplicate SHA256")
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

	found, err := fs.FindBySHA256(created.SHA256)
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

	found, err := fs.FindByNameAndSize(created.OriginalName, created.SizeBytes)
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

	root, err := fs.ListDirs(user.ID)
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
