package backup

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
)

type LastBackupResult struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	SizeBytes int64  `json:"size_bytes"`
	Error     string `json:"error,omitempty"`
}

type Scheduler struct {
	cfg           *config.Config
	db            *store.DB
	storage       *service.StorageService
	eventRecorder *service.EventRecorder
	lastResult    *LastBackupResult
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

var backupKeyPattern = regexp.MustCompile(`^backups/database/drive-backup-(\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2})\.db$`)

func NewScheduler(cfg *config.Config, db *store.DB, storage *service.StorageService, eventRecorder *service.EventRecorder) *Scheduler {
	return &Scheduler{
		cfg:           cfg,
		db:            db,
		storage:       storage,
		eventRecorder: eventRecorder,
		stopCh:        make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	if !s.cfg.Backup.Enabled {
		slog.Warn("backup is disabled, skipping scheduler start")
		return
	}
	if !s.storage.IsConnected() {
		slog.Warn("S3 is not connected, skipping backup scheduler start")
		return
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.RunBackup()

		ticker := time.NewTicker(time.Duration(s.cfg.Backup.IntervalH) * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.RunBackup()
			}
		}
	}()

	slog.Info("backup scheduler started", "interval_h", s.cfg.Backup.IntervalH)
}

func (s *Scheduler) Shutdown() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Scheduler) RunBackup() {
	startedAt := time.Now().UTC()
	timestamp := startedAt.Format("2006-01-02T15-04-05")

	tempPath := filepath.Join("/tmp/kilo", fmt.Sprintf("drive-backup-%s.db", timestamp))
	defer os.Remove(tempPath)

	duration := func() int64 { return time.Since(startedAt).Milliseconds() }

	if _, err := s.db.Exec(fmt.Sprintf("VACUUM INTO '%s'", tempPath)); err != nil {
		slog.Error("backup VACUUM INTO failed", "error", err)
		s.eventRecorder.Error("backup_failure", "VACUUM INTO failed", map[string]interface{}{
			"error":       err.Error(),
			"duration_ms": duration(),
		})
		s.mu.Lock()
		s.lastResult = &LastBackupResult{Status: "failure", Timestamp: startedAt.Format(time.RFC3339), Error: err.Error()}
		s.mu.Unlock()
		return
	}

	stat, err := os.Stat(tempPath)
	if err != nil {
		slog.Error("backup stat failed", "error", err)
		s.eventRecorder.Error("backup_failure", "Failed to stat backup file", map[string]interface{}{
			"error":       err.Error(),
			"duration_ms": duration(),
		})
		s.mu.Lock()
		s.lastResult = &LastBackupResult{Status: "failure", Timestamp: startedAt.Format(time.RFC3339), Error: err.Error()}
		s.mu.Unlock()
		return
	}
	sizeBytes := stat.Size()

	s3Key := fmt.Sprintf("backups/database/drive-backup-%s.db", timestamp)
	if err := s.storage.PutObject(s3Key, tempPath); err != nil {
		slog.Error("backup S3 upload failed", "error", err)
		s.eventRecorder.Error("backup_failure", "S3 upload failed", map[string]interface{}{
			"error":       err.Error(),
			"size_bytes":  sizeBytes,
			"duration_ms": duration(),
		})
		s.mu.Lock()
		s.lastResult = &LastBackupResult{Status: "failure", Timestamp: startedAt.Format(time.RFC3339), Error: err.Error()}
		s.mu.Unlock()
		return
	}

	s.eventRecorder.Info("backup_success", fmt.Sprintf("Database backup uploaded to S3 (%d bytes)", sizeBytes), map[string]interface{}{
		"size_bytes":  sizeBytes,
		"s3_key":      s3Key,
		"duration_ms": duration(),
	})

	s.mu.Lock()
	s.lastResult = &LastBackupResult{Status: "success", Timestamp: startedAt.Format(time.RFC3339), SizeBytes: sizeBytes}
	s.mu.Unlock()

	s.pruneOldBackups()

	slog.Info("backup completed", "s3_key", s3Key, "size_bytes", sizeBytes)
}

func (s *Scheduler) pruneOldBackups() {
	keys, err := s.storage.ListObjects("backups/database/")
	if err != nil {
		slog.Warn("backup retention: failed to list S3 objects", "error", err)
		return
	}

	cutoff := time.Now().UTC().Add(-time.Duration(s.cfg.Backup.RetentionDays) * 24 * time.Hour)
	pruned := 0

	for _, key := range keys {
		matches := backupKeyPattern.FindStringSubmatch(key)
		if len(matches) != 2 {
			continue
		}
		keyTime, err := time.Parse("2006-01-02T15-04-05", matches[1])
		if err != nil {
			continue
		}
		if keyTime.Before(cutoff) {
			if err := s.storage.DeleteObject(key); err != nil {
				slog.Warn("backup retention: failed to delete old backup", "key", key, "error", err)
			} else {
				pruned++
			}
		}
	}

	if pruned > 0 {
		s.eventRecorder.Info("backup_pruned", fmt.Sprintf("Pruned %d old backups (retention: %d days)", pruned, s.cfg.Backup.RetentionDays), map[string]interface{}{
			"pruned_count":   pruned,
			"retention_days": s.cfg.Backup.RetentionDays,
		})
		slog.Info("backup retention pruned", "count", pruned, "retention_days", s.cfg.Backup.RetentionDays)
	}
}

func (s *Scheduler) LastResult() *LastBackupResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.lastResult == nil {
		return nil
	}
	r := *s.lastResult
	return &r
}
