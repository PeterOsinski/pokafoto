package service

import (
	"log/slog"
	"time"

	"github.com/drive/drive/internal/store"
)

type EventRetention struct {
	store store.SystemEventsRepository
}

func NewEventRetention(store store.SystemEventsRepository) *EventRetention {
	return &EventRetention{store: store}
}

func (e *EventRetention) Start(stopCh <-chan struct{}) {
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-time.After(24 * time.Hour):
			}
			deleted, err := e.store.PurgeOlderThan(90 * 24 * time.Hour)
			if err != nil {
				slog.Warn("event retention purge failed", "error", err)
			} else if deleted > 0 {
				slog.Info("purged old system events", "deleted", deleted)
			}
		}
	}()
}
