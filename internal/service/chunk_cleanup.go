package service

import (
	"time"

	"github.com/drive/drive/internal/store"
)

type ChunkCleanup struct {
	chunkStore       store.ChunkRepository
	cleanupHours     int
	maxUploadAgeHours int
}

func NewChunkCleanup(chunkStore store.ChunkRepository, cleanupHours, maxUploadAgeHours int) *ChunkCleanup {
	if cleanupHours <= 0 {
		cleanupHours = 24
	}
	if maxUploadAgeHours <= 0 {
		maxUploadAgeHours = 48
	}
	return &ChunkCleanup{
		chunkStore:        chunkStore,
		cleanupHours:      cleanupHours,
		maxUploadAgeHours: maxUploadAgeHours,
	}
}

func (c *ChunkCleanup) Start(stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(60 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				c.chunkStore.DeleteAbandonedChunks(c.cleanupHours)
				c.chunkStore.CleanupOldUploads(c.maxUploadAgeHours)
			}
		}
	}()
}
