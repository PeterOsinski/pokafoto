package store

import (
	"testing"
	"time"

	"github.com/drive/drive/internal/model"
)

func TestSystemEventsStore_Create_shouldInsert(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	msg := "test event"
	event := &model.SystemEvent{
		EventType: "test_entry",
		Severity:  model.SeverityInfo,
		Message:   msg,
	}
	if err := s.Create(event); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if event.ID == "" {
		t.Error("expected ID to be set")
	}
	if event.CreatedAt == "" {
		t.Error("expected CreatedAt to be set")
	}

	events, total, err := s.List(10, 0, "test_entry", "", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 total, got %d", total)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType != "test_entry" {
		t.Errorf("expected test_entry, got %s", events[0].EventType)
	}
}

func TestSystemEventsStore_List_shouldFilterByType(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	s.Create(&model.SystemEvent{EventType: "type_a", Severity: model.SeverityInfo, Message: "a1"})
	s.Create(&model.SystemEvent{EventType: "type_b", Severity: model.SeverityInfo, Message: "b1"})
	s.Create(&model.SystemEvent{EventType: "type_a", Severity: model.SeverityInfo, Message: "a2"})

	events, total, err := s.List(10, 0, "type_a", "", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 total for type_a, got %d", total)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
	for _, e := range events {
		if e.EventType != "type_a" {
			t.Errorf("expected type_a, got %s", e.EventType)
		}
	}
}

func TestSystemEventsStore_List_shouldFilterBySeverity(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	s.Create(&model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "info msg"})
	s.Create(&model.SystemEvent{EventType: "test", Severity: model.SeverityError, Message: "error msg"})
	s.Create(&model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "info msg 2"})

	events, total, err := s.List(10, 0, "", "error", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 error, got %d", total)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
	if events[0].Severity != model.SeverityError {
		t.Errorf("expected error severity, got %s", events[0].Severity)
	}
}

func TestSystemEventsStore_List_shouldFilterByDateRange(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	now := time.Now().UTC()

	e1 := &model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "old"}
	e2 := &model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "new"}
	s.Create(e1)
	s.Create(e2)

	e1.CreatedAt = now.Add(-48 * time.Hour).Format(time.RFC3339)
	e2.CreatedAt = now.Format(time.RFC3339)
	s.db.Exec("UPDATE system_events SET created_at = ? WHERE id = ?", e1.CreatedAt, e1.ID)
	s.db.Exec("UPDATE system_events SET created_at = ? WHERE id = ?", e2.CreatedAt, e2.ID)

	dateFrom := now.Add(-24 * time.Hour).Format(time.RFC3339)
	events, total, err := s.List(10, 0, "", "", dateFrom, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 event in range, got %d (dateFrom=%s)", total, dateFrom)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestSystemEventsStore_List_shouldPaginateDescending(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	for i := 0; i < 5; i++ {
		s.Create(&model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "msg"})
	}

	events, total, err := s.List(3, 0, "", "", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 5 {
		t.Errorf("expected 5 total, got %d", total)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events (limit), got %d", len(events))
	}

	events2, _, err := s.List(3, 3, "", "", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events2) != 2 {
		t.Errorf("expected 2 events (offset 3), got %d", len(events2))
	}
}

func TestSystemEventsStore_EventCounts_shouldReturnGrouped(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	s.Create(&model.SystemEvent{EventType: "backup_success", Severity: model.SeverityInfo, Message: "b1"})
	s.Create(&model.SystemEvent{EventType: "backup_success", Severity: model.SeverityInfo, Message: "b2"})
	s.Create(&model.SystemEvent{EventType: "upload_error", Severity: model.SeverityError, Message: "e1"})

	counts, err := s.EventCounts()
	if err != nil {
		t.Fatalf("EventCounts: %v", err)
	}
	if counts["backup_success"] != 2 {
		t.Errorf("expected 2 backup_success, got %d", counts["backup_success"])
	}
	if counts["upload_error"] != 1 {
		t.Errorf("expected 1 upload_error, got %d", counts["upload_error"])
	}
}

func TestSystemEventsStore_PurgeOlderThan_shouldDelete(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	now := time.Now().UTC()

	eOld := &model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "old"}
	eNew := &model.SystemEvent{EventType: "test", Severity: model.SeverityInfo, Message: "new"}
	s.Create(eOld)
	s.Create(eNew)

	eOld.CreatedAt = now.Add(-120 * 24 * time.Hour).Format(time.RFC3339)
	eNew.CreatedAt = now.Format(time.RFC3339)
	s.db.Exec("UPDATE system_events SET created_at = ? WHERE id = ?", eOld.CreatedAt, eOld.ID)
	s.db.Exec("UPDATE system_events SET created_at = ? WHERE id = ?", eNew.CreatedAt, eNew.ID)

	deleted, err := s.PurgeOlderThan(90 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("PurgeOlderThan: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	_, total, _ := s.List(10, 0, "", "", "", "")
	if total != 1 {
		t.Errorf("expected 1 remaining, got %d", total)
	}
}

func TestSystemEventsStore_EventCounts_shouldReturnEmptyForNoEvents(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	counts, err := s.EventCounts()
	if err != nil {
		t.Fatalf("EventCounts: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %d entries", len(counts))
	}
}

func TestSystemEventsStore_List_shouldHandleCombinedFilters(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	s.Create(&model.SystemEvent{EventType: "type_a", Severity: model.SeverityError, Message: "error a"})
	s.Create(&model.SystemEvent{EventType: "type_a", Severity: model.SeverityInfo, Message: "info a"})
	s.Create(&model.SystemEvent{EventType: "type_b", Severity: model.SeverityError, Message: "error b"})

	events, total, err := s.List(10, 0, "type_a", "error", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 combined match, got %d", total)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestSystemEventsStore_List_shouldReturnEmpty(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)
	defer db.Close()

	s := NewSystemEventsStore(db)
	events, total, err := s.List(10, 0, "nonexistent", "", "", "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if len(events) != 0 {
		t.Errorf("expected empty slice, got %d", len(events))
	}
}
