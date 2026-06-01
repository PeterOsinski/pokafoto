package backup

import (
	"os"
	"testing"

	"github.com/drive/drive/internal/config"
	"github.com/drive/drive/internal/service"
	"github.com/drive/drive/internal/store"
)

func TestBackup_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Backup.Enabled = false

	db := store.OpenTestDB(t)
	defer db.Close()

	storage, _ := service.NewStorageService(cfg)
	eventRecorder := service.NewEventRecorder(db)

	sched := NewScheduler(cfg, db, storage, eventRecorder)
	sched.Start()
	sched.Shutdown()
}

func TestBackup_NoS3(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Backup.Enabled = true

	db := store.OpenTestDB(t)
	defer db.Close()

	storage, _ := service.NewStorageService(cfg)
	eventRecorder := service.NewEventRecorder(db)

	sched := NewScheduler(cfg, db, storage, eventRecorder)
	sched.Start()
	sched.Shutdown()
}

func TestBackup_RunBackup_success(t *testing.T) {
	if os.Getenv("DRIVE_BACKUP_INTEGRATION") != "1" {
		t.Skip("skipping backup test without S3; set DRIVE_BACKUP_INTEGRATION=1")
	}

	cfg := config.Load()

	db := store.OpenTestDB(t)
	defer db.Close()

	storage, err := service.NewStorageService(cfg)
	if err != nil {
		t.Skipf("S3 not available: %v", err)
	}

	eventRecorder := service.NewEventRecorder(db)

	sched := NewScheduler(cfg, db, storage, eventRecorder)
	sched.RunBackup()

	result := sched.LastResult()
	if result == nil {
		t.Fatal("expected last result after backup")
	}
	if result.Status != "success" {
		t.Errorf("expected success, got %s: %s", result.Status, result.Error)
	}
	if result.SizeBytes <= 0 {
		t.Errorf("expected positive size, got %d", result.SizeBytes)
	}

	events := store.NewSystemEventsStore(db)
	allEvents, total, _ := events.List(10, 0, "backup_success", "", "", "")
	if total < 1 {
		t.Error("expected backup_success event in system_events")
	}
	_ = allEvents
}

func TestBackup_VacuumCreatesFile(t *testing.T) {
	db, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	defer db.Close()

	tempPath := "/tmp/kilo/drive-backup-test.db"
	os.MkdirAll("/tmp/kilo", 0755)
	defer os.Remove(tempPath)

	if _, err := db.Exec("VACUUM INTO '" + tempPath + "'"); err != nil {
		t.Fatalf("VACUUM INTO failed: %v", err)
	}

	stat, err := os.Stat(tempPath)
	if err != nil {
		t.Fatalf("stat backup file: %v", err)
	}
	if stat.Size() == 0 {
		t.Error("expected non-zero backup file size")
	}
}

func TestBackup_LastResult_initialNil(t *testing.T) {
	cfg := config.DefaultConfig()
	db := store.OpenTestDB(t)
	defer db.Close()

	storage, _ := service.NewStorageService(cfg)
	eventRecorder := service.NewEventRecorder(db)

	sched := NewScheduler(cfg, db, storage, eventRecorder)
	if sched.LastResult() != nil {
		t.Error("expected nil last result before any backup")
	}
}
