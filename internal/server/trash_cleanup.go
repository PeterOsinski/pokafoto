package server

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const trashCleanupChunkSize = 100

func (s *Server) startTrashCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupExpiredTrash()
		}
	}
}

func (s *Server) cleanupExpiredTrash() {
	expiration := time.Duration(s.cfg.TrashExpirationDays) * 24 * time.Hour
	cutoff := time.Now().UTC().Add(-expiration).Format(time.RFC3339)

	totalDeleted := 0
	for {
		files, err := s.fileStore.GetExpiredFiles(cutoff, trashCleanupChunkSize)
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
			originalPath := filepath.Join(s.cfg.OriginalsDir(), f.UserID, f.Filename)
			os.Remove(originalPath)
			thumbDir := filepath.Join(s.cfg.ThumbnailsDir(), f.ID)
			os.RemoveAll(thumbDir)
			s.enqueueS3Deletion(f.ID, f.UserID, f.Filename)
		}

		if err := s.fileStore.PermanentDeleteByIDs(ids); err != nil {
			slog.Warn("trash cleanup failed to delete rows", "error", err)
			return
		}
		totalDeleted += len(ids)
	}

	if totalDeleted > 0 {
		slog.Info("trash cleanup completed", "expired_count", totalDeleted)
	}
}
