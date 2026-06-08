package service

import (
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
	"github.com/minio/minio-go/v7"
)

type mockStorageProvider struct {
	deleteOrigCount atomic.Int64
	deleteThumbCount atomic.Int64
}

func (m *mockStorageProvider) PutObject(key string, filePath string) error                { return nil }
func (m *mockStorageProvider) GetObject(key string, destPath string) error                { return nil }
func (m *mockStorageProvider) GetObjectStream(key string) (io.ReadCloser, error)          { return nil, nil }
func (m *mockStorageProvider) IsConnected() bool                                           { return true }
func (m *mockStorageProvider) DeleteObject(key string) error                               { return nil }
func (m *mockStorageProvider) DeleteOriginal(userID, filename string) error {
	m.deleteOrigCount.Add(1)
	return nil
}
func (m *mockStorageProvider) DeleteThumbnail(fileID, size, format string) error {
	m.deleteThumbCount.Add(1)
	return nil
}
func (m *mockStorageProvider) ListObjects(prefix string) ([]string, error) { return nil, nil }
func (m *mockStorageProvider) Client() *minio.Client                        { return nil }

type mockThumbnailRefRepo struct{}

func (m *mockThumbnailRefRepo) Create(thumb *model.Thumbnail) error                                 { return nil }
func (m *mockThumbnailRefRepo) FindByFileIDAndSize(fileID string, size model.ThumbnailSize) (*model.Thumbnail, error) {
	return nil, nil
}
func (m *mockThumbnailRefRepo) FindThumbnailRefsByFileID(fileID string) ([]store.ThumbnailRef, error) {
	return []store.ThumbnailRef{{Size: "sm", Format: "jpg"}}, nil
}
func (m *mockThumbnailRefRepo) CountByFileID(fileID string) (int, error)                           { return 0, nil }
func (m *mockThumbnailRefRepo) SetS3Key(fileID string, size model.ThumbnailSize, s3Key string) error { return nil }
func (m *mockThumbnailRefRepo) TotalSize() (int64, error)                                           { return 0, nil }
func (m *mockThumbnailRefRepo) Breakdown() ([]store.ThumbnailBreakdown, error)                      { return nil, nil }
func (m *mockThumbnailRefRepo) BreakdownByUser(userID string) ([]store.ThumbnailBreakdown, error) { return nil, nil }

func TestS3DeletionPool_Enqueue_shouldProcessTask(t *testing.T) {
	t.Parallel()
	mockStorage := &mockStorageProvider{}
	mockThumb := &mockThumbnailRefRepo{}
	p := NewS3DeletionPool(mockStorage, mockThumb)

	task := &S3DeleteTask{
		FileID:   "file-1",
		UserID:   "user-1",
		Filename: "test.jpg",
		Thumbs:   []S3ThumbItem{{Size: "sm", Format: "jpg"}},
	}
	p.Enqueue(task)

	time.Sleep(100 * time.Millisecond)

	if mockStorage.deleteOrigCount.Load() != 1 {
		t.Errorf("expected 1 DeleteOriginal call, got %d", mockStorage.deleteOrigCount.Load())
	}
	if mockStorage.deleteThumbCount.Load() != 1 {
		t.Errorf("expected 1 DeleteThumbnail call, got %d", mockStorage.deleteThumbCount.Load())
	}
	if p.PendingCount() != 0 {
		t.Errorf("expected 0 pending, got %d", p.PendingCount())
	}

	p.Shutdown()
}

func TestS3DeletionPool_Shutdown_shouldStopWorkers(t *testing.T) {
	t.Parallel()
	mockStorage := &mockStorageProvider{}
	mockThumb := &mockThumbnailRefRepo{}
	p := NewS3DeletionPool(mockStorage, mockThumb)
	p.Shutdown()
}
