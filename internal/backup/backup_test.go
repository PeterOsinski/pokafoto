package backup

import (
	"testing"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
)

func setupTestScheduler(t *testing.T) (*Scheduler, *store.DB, *service.StorageService) {
	t.Helper()

	cfg := config.DefaultConfig()
	cfg.Storage.Local.Path = t.TempDir()

	db := store.OpenTestDB(t)
	fs := service.NewRealFS()
	storage, _ := service.NewStorageService(cfg, fs)
	recorder := service.NewEventRecorder(db)

	s := NewScheduler(cfg, db, storage, recorder, fs)
	return s, db, storage
}

func TestScheduler_New_shouldInitializeFields(t *testing.T) {
	t.Parallel()

	cfg := config.DefaultConfig()
	db := store.OpenTestDB(t)
	fs := service.NewRealFS()
	storage, _ := service.NewStorageService(cfg, fs)
	recorder := service.NewEventRecorder(db)

	s := NewScheduler(cfg, db, storage, recorder, fs)

	if s.cfg != cfg {
		t.Error("cfg not set")
	}
	if s.db != db {
		t.Error("db not set")
	}
	if s.storage != storage {
		t.Error("storage not set")
	}
	if s.fs != fs {
		t.Error("fs not set")
	}
	if s.eventRecorder != recorder {
		t.Error("eventRecorder not set")
	}
	if s.stopCh == nil {
		t.Error("stopCh should be initialized")
	}
}

func TestScheduler_Start_backupDisabled_shouldReturnEarly(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)
	s.cfg.Backup.Enabled = false

	s.Start()

	select {
	case <-s.stopCh:
		t.Error("stopCh should not be closed when Start returns early")
	default:
	}
}

func TestScheduler_Start_s3NotConnected_shouldReturnEarly(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)
	s.cfg.Backup.Enabled = true

	s.Start()

	select {
	case <-s.stopCh:
		t.Error("stopCh should not be closed when Start returns early")
	default:
	}
}

func TestScheduler_LastResult_returnsNilWhenNoResult(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)

	result := s.LastResult()
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestScheduler_LastResult_returnsCopy(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)

	s.mu.Lock()
	s.lastResult = &LastBackupResult{Status: "success", SizeBytes: 1024}
	s.mu.Unlock()

	result := s.LastResult()
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Status != "success" {
		t.Errorf("expected success, got %s", result.Status)
	}
	if result.SizeBytes != 1024 {
		t.Errorf("expected 1024, got %d", result.SizeBytes)
	}
}

func TestScheduler_Shutdown_shouldCloseStopCh(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)

	s.Shutdown()

	select {
	case _, ok := <-s.stopCh:
		if ok {
			t.Error("stopCh should be closed")
		}
	default:
		t.Error("stopCh should be closed after Shutdown")
	}
}

func TestScheduler_pruneOldBackups_retentionDaysZero_shouldNoop(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)
	s.cfg.Backup.RetentionDays = 0

	s.pruneOldBackups()
}

func TestScheduler_pruneOldBackups_noS3Client_shouldNotPanic(t *testing.T) {
	t.Parallel()

	s, _, _ := setupTestScheduler(t)
	s.cfg.Backup.RetentionDays = 7

	s.pruneOldBackups()
}

func TestScheduler_RunBackup_noS3Client_shouldComplete(t *testing.T) {
	t.Parallel()

	s, db, _ := setupTestScheduler(t)
	s.cfg.Backup.Enabled = true

	if err := s.fs.MkdirAll("/tmp/kilo", 0755); err != nil {
		t.Fatalf("create /tmp/kilo: %v", err)
	}

	db.Exec("CREATE TABLE IF NOT EXISTS test_table (id INTEGER PRIMARY KEY)")

	s.RunBackup()

	result := s.LastResult()
	if result == nil {
		t.Fatal("expected a result after RunBackup")
	}
	if result.Status != "success" {
		t.Errorf("expected success, got %s (error: %s)", result.Status, result.Error)
	}
	if result.SizeBytes <= 0 {
		t.Error("expected positive size_bytes")
	}
}
