package service

import (
	"time"

	"github.com/drive/drive/internal/store"
)

type FolderPasswordCleanup struct {
	store store.FolderPasswordRepository
}

func NewFolderPasswordCleanup(store store.FolderPasswordRepository) *FolderPasswordCleanup {
	return &FolderPasswordCleanup{store: store}
}

func (c *FolderPasswordCleanup) Start(stopCh <-chan struct{}) {
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case <-time.After(5 * time.Minute):
			}
			c.store.DeleteExpired()
		}
	}()
}
