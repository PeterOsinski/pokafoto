package service

import (
	"sync/atomic"
	"testing"
	"time"
)

type mockChunkRepo struct {
	deleteAbandonedCount atomic.Int64
	cleanupOldCount      atomic.Int64
}

func (m *mockChunkRepo) CreateChunkRecord(uploadID string, index int, size, offset int64, sha256hex, tempPath string) error {
	return nil
}
func (m *mockChunkRepo) GetStoredChunks(uploadID string) ([]int, error)                   { return nil, nil }
func (m *mockChunkRepo) GetStoredChunkCount(uploadID string) (int, error)                  { return 0, nil }
func (m *mockChunkRepo) GetChunkPath(uploadID string, index int) (string, error)            { return "", nil }
func (m *mockChunkRepo) FindMissingChunks(uploadID string, totalChunks int) ([]int, error) { return nil, nil }
func (m *mockChunkRepo) AssembleFile(uploadID string, totalChunks int, destPath string) (string, error) {
	return "", nil
}
func (m *mockChunkRepo) DeleteChunks(uploadID string) error      { return nil }
func (m *mockChunkRepo) CleanupOrphanedTempFiles(uploadID string) {}
func (m *mockChunkRepo) DeleteAbandonedChunks(maxAgeHours int) (int64, error) {
	m.deleteAbandonedCount.Add(1)
	return 0, nil
}
func (m *mockChunkRepo) CleanupOldUploads(maxAgeHours int) ([]string, error) {
	m.cleanupOldCount.Add(1)
	return nil, nil
}

func TestChunkCleanup_Start_shouldStartWithoutPanic(t *testing.T) {
	t.Parallel()
	mockStore := &mockChunkRepo{}
	c := NewChunkCleanup(mockStore, 24, 48)

	stopCh := make(chan struct{})
	c.Start(stopCh)
	close(stopCh)

	time.Sleep(10 * time.Millisecond)
}

func TestChunkCleanup_New_defaultsWhenZero(t *testing.T) {
	t.Parallel()
	mockStore := &mockChunkRepo{}
	c := NewChunkCleanup(mockStore, 0, 0)
	if c.cleanupHours != 24 {
		t.Errorf("expected cleanupHours=24, got %d", c.cleanupHours)
	}
	if c.maxUploadAgeHours != 48 {
		t.Errorf("expected maxUploadAgeHours=48, got %d", c.maxUploadAgeHours)
	}
}
