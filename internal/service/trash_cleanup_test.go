package service

import (
	"fmt"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
)

type mockFileRepo struct {
	getExpiredFiles    func(cutoff string, limit int) ([]store.ExpiredFile, error)
	permanentDeleteIDs func(ids []string) error
}

func (m *mockFileRepo) Create(file *model.File) error                        { return nil }
func (m *mockFileRepo) FindByID(id string) (*model.File, error)               { return nil, nil }
func (m *mockFileRepo) FindBySHA256(userID, sha256 string) (*model.File, error) { return nil, nil }
func (m *mockFileRepo) FindByNameAndSize(userID, name string, size int64) (*model.File, error) {
	return nil, nil
}
func (m *mockFileRepo) FindByNameAndSizeBatch(userID string, nameSizes []store.FileRecord) ([]*model.File, error) {
	return nil, nil
}
func (m *mockFileRepo) List(opts store.FileListOptions) ([]*model.File, string, int, error) {
	return nil, "", 0, nil
}
func (m *mockFileRepo) SoftDelete(id string) error      { return nil }
func (m *mockFileRepo) PermanentDelete(id string) error { return nil }
func (m *mockFileRepo) Stats(userID string) (*store.StatsResult, error) {
	return nil, nil
}
func (m *mockFileRepo) ListDirs(userID string, allFolders bool) (*store.DirEntry, error) {
	return nil, nil
}
func (m *mockFileRepo) SearchEnhanced(opts store.SearchOptions) (*store.SearchResult, map[string]string, error) {
	return nil, nil, nil
}
func (m *mockFileRepo) Search(userID, query string, limit int) (*store.SearchResult, error) {
	return nil, nil
}
func (m *mockFileRepo) Timeline(userID, granularity string) ([]store.TimelineGroup, error) {
	return nil, nil
}
func (m *mockFileRepo) BatchSoftDelete(userID string, ids []string) error          { return nil }
func (m *mockFileRepo) BatchMove(userID string, ids []string, folderID *string) error { return nil }
func (m *mockFileRepo) BatchCopy(userID string, ids []string, folderID *string) ([]*model.File, error) {
	return nil, nil
}
func (m *mockFileRepo) FindPhotosMissingThumbnails() ([]*model.File, error) { return nil, nil }
func (m *mockFileRepo) CountPhotosMissingThumbnailPreview() (int, error)     { return 0, nil }
func (m *mockFileRepo) Restore(id string) error                              { return nil }
func (m *mockFileRepo) BatchRestore(userID string, ids []string) error        { return nil }
func (m *mockFileRepo) ListTrash(opts store.FileListOptions) ([]*model.File, string, int, error) {
	return nil, "", 0, nil
}
func (m *mockFileRepo) TrashStats(userID string) (*store.TrashStatsResult, error) { return nil, nil }
func (m *mockFileRepo) GetExpiredFiles(cutoff string, limit int) ([]store.ExpiredFile, error) {
	if m.getExpiredFiles != nil {
		return m.getExpiredFiles(cutoff, limit)
	}
	return nil, nil
}
func (m *mockFileRepo) PermanentDeleteByIDs(ids []string) error {
	if m.permanentDeleteIDs != nil {
		return m.permanentDeleteIDs(ids)
	}
	return nil
}
func (m *mockFileRepo) BatchPermanentDelete(userID string, ids []string) error { return nil }
func (m *mockFileRepo) UpdateSizeAndHash(id string, sizeBytes int64, sha256 string) error {
	return nil
}
func (m *mockFileRepo) ListFilesByFolderID(folderID, cursor string, limit int) ([]*model.File, string, int, error) {
	return nil, "", 0, nil
}
func (m *mockFileRepo) Rename(id, userID, newName string) error                         { return nil }
func (m *mockFileRepo) SoftDeleteByFolderIDs(userID string, folderIDs []string) (int64, error) {
	return 0, nil
}
func (m *mockFileRepo) ListTrashFiles(userID string, ids []string) ([]store.ExpiredFile, error) {
	return nil, nil
}
func (m *mockFileRepo) ListAllTrashFiles(userID string) ([]store.ExpiredFile, error) {
	return nil, nil
}
func (m *mockFileRepo) AdminFileBreakdown() (*store.AdminFileBreakdown, error)       { return nil, nil }
func (m *mockFileRepo) AdminFileBreakdownByUser(userID string) (*store.AdminFileBreakdown, error) {
	return nil, nil
}

func TestTrashCleanup_New_shouldInitializeFields(t *testing.T) {
	t.Parallel()
	fs := NewMockFS()
	tc := NewTrashCleanup(nil, fs, "/data/originals", "/data/thumbnails", 30, nil)

	if tc.originalsDir != "/data/originals" {
		t.Errorf("expected /data/originals, got %s", tc.originalsDir)
	}
	if tc.thumbnailsDir != "/data/thumbnails" {
		t.Errorf("expected /data/thumbnails, got %s", tc.thumbnailsDir)
	}
	if tc.trashExpirationDays != 30 {
		t.Errorf("expected 30, got %d", tc.trashExpirationDays)
	}
}

func TestTrashCleanup_cleanupExpiredTrash_noExpiredFiles(t *testing.T) {
	t.Parallel()
	fs := NewMockFS()
	mockRepo := &mockFileRepo{
		getExpiredFiles: func(cutoff string, limit int) ([]store.ExpiredFile, error) {
			return nil, nil
		},
	}

	tc := NewTrashCleanup(mockRepo, fs, "/data/originals", "/data/thumbnails", 30, nil)
	tc.cleanupExpiredTrash()
}

func TestTrashCleanup_cleanupExpiredTrash_singleFile(t *testing.T) {
	t.Parallel()
	fs := NewMockFS()
	fs.AddFile("/data/originals/user1/photo.jpg", make([]byte, 100))

	deletedIDs := make([]string, 0)
	mockRepo := &mockFileRepo{
		getExpiredFiles: func(cutoff string, limit int) ([]store.ExpiredFile, error) {
			if len(deletedIDs) == 0 {
				return []store.ExpiredFile{
					{ID: "file1", UserID: "user1", Filename: "photo.jpg"},
				}, nil
			}
			return nil, nil
		},
		permanentDeleteIDs: func(ids []string) error {
			deletedIDs = append(deletedIDs, ids...)
			return nil
		},
	}

	onDeleteCalled := false
	tc := NewTrashCleanup(mockRepo, fs, "/data/originals", "/data/thumbnails", 30, func(fileID, userID, filename string) {
		onDeleteCalled = true
		if fileID != "file1" {
			t.Errorf("expected file1, got %s", fileID)
		}
	})

	tc.cleanupExpiredTrash()

	if len(deletedIDs) != 1 || deletedIDs[0] != "file1" {
		t.Errorf("expected file1 to be deleted, got %v", deletedIDs)
	}
	if !onDeleteCalled {
		t.Error("onDelete callback should have been called")
	}
}

func TestTrashCleanup_cleanupExpiredTrash_getFilesError(t *testing.T) {
	t.Parallel()
	fs := NewMockFS()
	mockRepo := &mockFileRepo{
		getExpiredFiles: func(cutoff string, limit int) ([]store.ExpiredFile, error) {
			return nil, fmt.Errorf("db error")
		},
	}

	tc := NewTrashCleanup(mockRepo, fs, "/data/originals", "/data/thumbnails", 30, nil)
	tc.cleanupExpiredTrash()
}

func TestTrashCleanup_cleanupExpiredTrash_deleteRowsError(t *testing.T) {
	t.Parallel()
	fs := NewMockFS()
	mockRepo := &mockFileRepo{
		getExpiredFiles: func(cutoff string, limit int) ([]store.ExpiredFile, error) {
			return []store.ExpiredFile{
				{ID: "file1", UserID: "user1", Filename: "photo.jpg"},
			}, nil
		},
		permanentDeleteIDs: func(ids []string) error {
			return fmt.Errorf("delete error")
		},
	}

	tc := NewTrashCleanup(mockRepo, fs, "/data/originals", "/data/thumbnails", 30, nil)
	tc.cleanupExpiredTrash()
}

func TestTrashCleanup_cleanupExpiredTrash_batchesFiles(t *testing.T) {
	t.Parallel()
	fs := NewMockFS()
	callCount := 0

	mockRepo := &mockFileRepo{
		getExpiredFiles: func(cutoff string, limit int) ([]store.ExpiredFile, error) {
			callCount++
			if callCount <= 2 {
				files := make([]store.ExpiredFile, 5)
				for i := range files {
					idx := (callCount-1)*5 + i
					files[i] = store.ExpiredFile{ID: fmt.Sprintf("file%d", idx), UserID: "user1", Filename: fmt.Sprintf("img%d.jpg", idx)}
				}
				return files, nil
			}
			return nil, nil
		},
	}

	tc := NewTrashCleanup(mockRepo, fs, "/data/originals", "/data/thumbnails", 30, nil)
	tc.cleanupExpiredTrash()

	if callCount != 3 {
		t.Errorf("expected 3 calls (2 batches + 1 empty), got %d", callCount)
	}
}
