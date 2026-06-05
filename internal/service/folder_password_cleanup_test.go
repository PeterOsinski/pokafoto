package service

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

type mockFolderPasswordRepo struct {
	deleteExpiredCount atomic.Int64
}

func (m *mockFolderPasswordRepo) Create(folderID, passwordHash, passwordHint string, expiresAt time.Time) (*model.FolderPassword, error) {
	return nil, nil
}
func (m *mockFolderPasswordRepo) FindByFolderID(folderID string) (*model.FolderPassword, error) { return nil, nil }
func (m *mockFolderPasswordRepo) DeleteByFolderID(folderID string) error                      { return nil }
func (m *mockFolderPasswordRepo) DeleteExpired() (int64, error) {
	m.deleteExpiredCount.Add(1)
	return 0, nil
}

func TestFolderPasswordCleanup_Start_shouldStartWithoutPanic(t *testing.T) {
	mockStore := &mockFolderPasswordRepo{}
	c := NewFolderPasswordCleanup(mockStore)

	stopCh := make(chan struct{})
	c.Start(stopCh)
	close(stopCh)

	time.Sleep(10 * time.Millisecond)
}

func TestFolderPasswordCleanup_New_shouldCreateWithStore(t *testing.T) {
	mockStore := &mockFolderPasswordRepo{}
	c := NewFolderPasswordCleanup(mockStore)
	if c == nil {
		t.Fatal("expected non-nil FolderPasswordCleanup")
	}
	if c.store != mockStore {
		t.Error("expected store to be set")
	}
}
