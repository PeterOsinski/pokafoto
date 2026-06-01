package service

import (
	"encoding/json"
	"log/slog"

	"github.com/drive/drive/internal/model"
	"github.com/drive/drive/internal/store"
)

type EventRecorder struct {
	store *store.SystemEventsStore
}

func NewEventRecorder(db *store.DB) *EventRecorder {
	return &EventRecorder{store: store.NewSystemEventsStore(db)}
}

func (r *EventRecorder) Record(eventType string, severity model.EventSeverity, message string, metadata map[string]interface{}) {
	if r == nil || r.store == nil {
		return
	}
	event := &model.SystemEvent{
		EventType: eventType,
		Severity:  severity,
		Message:   message,
	}

	if len(metadata) > 0 {
		raw, err := json.Marshal(metadata)
		if err != nil {
			slog.Warn("event recorder: failed to marshal metadata", "event_type", eventType, "error", err)
		} else {
			s := string(raw)
			event.Metadata = &s
		}
	}

	if err := r.store.Create(event); err != nil {
		slog.Warn("event recorder: failed to persist event", "event_type", eventType, "error", err)
	}
}

func (r *EventRecorder) Info(eventType, message string, metadata map[string]interface{}) {
	r.Record(eventType, model.SeverityInfo, message, metadata)
}

func (r *EventRecorder) Warn(eventType, message string, metadata map[string]interface{}) {
	r.Record(eventType, model.SeverityWarning, message, metadata)
}

func (r *EventRecorder) Error(eventType, message string, metadata map[string]interface{}) {
	r.Record(eventType, model.SeverityError, message, metadata)
}
