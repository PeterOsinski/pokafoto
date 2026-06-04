package service

import (
	"strings"
	"testing"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
)

func TestEventRecorder_Record_nilReceiver(t *testing.T) {
	var r *EventRecorder
	r.Record("test", model.SeverityInfo, "message", nil)
}

func TestEventRecorder_Record_validEvent(t *testing.T) {
	db := store.OpenTestDB(t)
	r := NewEventRecorder(db)

	r.Record("test_event", model.SeverityInfo, "test message", map[string]interface{}{
		"key": "value",
	})

	eventsStore := store.NewSystemEventsStore(db)
	events, total, err := eventsStore.List(1, 0, "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("expected 1 event, got %d", total)
	}
	if events[0].EventType != "test_event" {
		t.Errorf("expected event_type test_event, got %s", events[0].EventType)
	}
	if events[0].Message != "test message" {
		t.Errorf("expected message 'test message', got %s", events[0].Message)
	}
	if events[0].Severity != model.SeverityInfo {
		t.Errorf("expected severity info, got %s", events[0].Severity)
	}
	if events[0].Metadata == nil {
		t.Fatal("expected metadata not nil")
	}
	if !strings.Contains(*events[0].Metadata, `"key":"value"`) {
		t.Errorf("expected metadata to contain key:value, got %s", *events[0].Metadata)
	}
}

func TestEventRecorder_Record_noMetadata(t *testing.T) {
	db := store.OpenTestDB(t)
	r := NewEventRecorder(db)

	r.Record("test_event", model.SeverityInfo, "no metadata", nil)

	eventsStore := store.NewSystemEventsStore(db)
	events, total, _ := eventsStore.List(1, 0, "", "", "", "")
	if total != 1 {
		t.Fatalf("expected 1 event, got %d", total)
	}
	if events[0].Metadata != nil {
		t.Errorf("expected nil metadata, got %s", *events[0].Metadata)
	}
}

func TestEventRecorder_Info(t *testing.T) {
	db := store.OpenTestDB(t)
	r := NewEventRecorder(db)

	r.Info("info_event", "info message", map[string]interface{}{"a": 1})

	eventsStore := store.NewSystemEventsStore(db)
	events, total, _ := eventsStore.List(1, 0, "", "", "", "")
	if total != 1 {
		t.Fatalf("expected 1 event, got %d", total)
	}
	if events[0].Severity != model.SeverityInfo {
		t.Errorf("expected severity info, got %s", events[0].Severity)
	}
}

func TestEventRecorder_Warn(t *testing.T) {
	db := store.OpenTestDB(t)
	r := NewEventRecorder(db)

	r.Warn("warn_event", "warn message", nil)

	eventsStore := store.NewSystemEventsStore(db)
	events, total, _ := eventsStore.List(1, 0, "", "", "", "")
	if total != 1 {
		t.Fatalf("expected 1 event, got %d", total)
	}
	if events[0].Severity != model.SeverityWarning {
		t.Errorf("expected severity warning, got %s", events[0].Severity)
	}
}

func TestEventRecorder_Error(t *testing.T) {
	db := store.OpenTestDB(t)
	r := NewEventRecorder(db)

	r.Error("error_event", "error message", nil)

	eventsStore := store.NewSystemEventsStore(db)
	events, total, _ := eventsStore.List(1, 0, "", "", "", "")
	if total != 1 {
		t.Fatalf("expected 1 event, got %d", total)
	}
	if events[0].Severity != model.SeverityError {
		t.Errorf("expected severity error, got %s", events[0].Severity)
	}
}

func TestEventRecorder_Record_nilStore(t *testing.T) {
	r := &EventRecorder{store: nil}
	r.Record("test", model.SeverityInfo, "msg", nil)
}
