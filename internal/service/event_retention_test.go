package service

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

type mockSystemEventsRepo struct {
	purgeCount atomic.Int64
}

func (m *mockSystemEventsRepo) Create(event *model.SystemEvent) error { return nil }
func (m *mockSystemEventsRepo) List(limit, offset int, eventType, severity, dateFrom, dateTo string) ([]model.SystemEvent, int, error) {
	return nil, 0, nil
}
func (m *mockSystemEventsRepo) EventCounts() (map[string]int, error) { return nil, nil }
func (m *mockSystemEventsRepo) PurgeOlderThan(age time.Duration) (int64, error) {
	m.purgeCount.Add(1)
	return 1, nil
}

func TestEventRetention_Start_shouldStartWithoutPanic(t *testing.T) {
	t.Parallel()
	mockStore := &mockSystemEventsRepo{}
	r := NewEventRetention(mockStore)

	stopCh := make(chan struct{})
	r.Start(stopCh)
	close(stopCh)

	time.Sleep(10 * time.Millisecond)
}

func TestEventRetention_New_shouldCreateWithStore(t *testing.T) {
	t.Parallel()
	mockStore := &mockSystemEventsRepo{}
	r := NewEventRetention(mockStore)
	if r == nil {
		t.Fatal("expected non-nil EventRetention")
	}
	if r.store != mockStore {
		t.Error("expected store to be set")
	}
}
