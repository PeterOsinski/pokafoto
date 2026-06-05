package service

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/drive/drive/internal/store"
)

const trashCleanupChunkSize = 100

type TrashCleanup struct {
	fileStore           store.FileRepository
	fs                  FileSystem
	originalsDir        string
	thumbnailsDir       string
	trashExpirationDays int
	onDelete            func(fileID, userID, filename string)
}

func NewTrashCleanup(fileStore store.FileRepository, fs FileSystem, originalsDir, thumbnailsDir string, trashExpirationDays int, onDelete func(fileID, userID, filename string)) *TrashCleanup {
	return &TrashCleanup{
		fileStore:           fileStore,
		fs:                  fs,
		originalsDir:        originalsDir,
		thumbnailsDir:       thumbnailsDir,
		trashExpirationDays: trashExpirationDays,
		onDelete:            onDelete,
	}
}

func (t *TrashCleanup) Start(stopCh <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				t.cleanupExpiredTrash()
			}
		}
	}()
}

func (t *TrashCleanup) cleanupExpiredTrash() {
	expiration := time.Duration(t.trashExpirationDays) * 24 * time.Hour
	cutoff := time.Now().UTC().Add(-expiration).Format(time.RFC3339)

	totalDeleted := 0
	for {
		files, err := t.fileStore.GetExpiredFiles(cutoff, trashCleanupChunkSize)
		if err != nil {
			slog.Warn("trash cleanup failed to get expired files", "error", err)
			return
		}
		if len(files) == 0 {
			break
		}

		ids := make([]string, len(files))
		for i, f := range files {
			ids[i] = f.ID
			t.fs.Remove(filepath.Join(t.originalsDir, f.UserID, f.Filename))
			t.fs.RemoveAll(filepath.Join(t.thumbnailsDir, f.ID))
			if t.onDelete != nil {
				t.onDelete(f.ID, f.UserID, f.Filename)
			}
		}

		if err := t.fileStore.PermanentDeleteByIDs(ids); err != nil {
			slog.Warn("trash cleanup failed to delete rows", "error", err)
			return
		}
		totalDeleted += len(ids)
	}

	if totalDeleted > 0 {
		slog.Info("trash cleanup completed", "expired_count", totalDeleted)
	}
}
