package config

import (
	"os"
	"testing"
)

func TestDefaultConfig_shouldSetExpectedDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Database.Path != "./data/drive.db" {
		t.Errorf("expected ./data/drive.db, got %s", cfg.Database.Path)
	}
	if cfg.Auth.SessionDurationH != 72 {
		t.Errorf("expected 72h, got %d", cfg.Auth.SessionDurationH)
	}
	if !cfg.Auth.AllowRegistration {
		t.Error("expected registration allowed by default")
	}
	if cfg.Upload.ConcurrentWorkers != 4 {
		t.Errorf("expected 4 workers, got %d", cfg.Upload.ConcurrentWorkers)
	}
	if cfg.Upload.MaxFileSizeMB != 10240 {
		t.Errorf("expected 10240 max file size, got %d", cfg.Upload.MaxFileSizeMB)
	}
	if cfg.Storage.S3.Enabled {
		t.Error("expected S3 disabled by default")
	}
}

func TestConfig_EnvOverridesPort(t *testing.T) {
	os.Setenv("DRIVE_PORT", "9090")
	defer os.Unsetenv("DRIVE_PORT")

	cfg := Load()
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
}

func TestConfig_EnvOverridesJWTSecret(t *testing.T) {
	os.Setenv("DRIVE_JWT_SECRET", "my-secret")
	defer os.Unsetenv("DRIVE_JWT_SECRET")

	cfg := Load()
	if cfg.Auth.JWTSecret != "my-secret" {
		t.Errorf("expected jwt secret 'my-secret', got %q", cfg.Auth.JWTSecret)
	}
}

func TestConfig_EnvOverridesStoragePath(t *testing.T) {
	os.Setenv("DRIVE_STORAGE_PATH", "/custom/data")
	defer os.Unsetenv("DRIVE_STORAGE_PATH")

	cfg := Load()
	if cfg.Storage.Local.Path != "/custom/data" {
		t.Errorf("expected /custom/data, got %s", cfg.Storage.Local.Path)
	}
}

func TestConfig_EnvOverridesDBPath(t *testing.T) {
	os.Setenv("DRIVE_DB_PATH", "/custom/db.db")
	defer os.Unsetenv("DRIVE_DB_PATH")

	cfg := Load()
	if cfg.Database.Path != "/custom/db.db" {
		t.Errorf("expected /custom/db.db, got %s", cfg.Database.Path)
	}
}

func TestConfig_EnvOverridesS3Enabled(t *testing.T) {
	os.Setenv("DRIVE_S3_ENABLED", "true")
	defer os.Unsetenv("DRIVE_S3_ENABLED")

	cfg := Load()
	if !cfg.Storage.S3.Enabled {
		t.Error("expected S3 enabled")
	}
}

func TestConfig_StoragePath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.Local.Path = "/data"

	got := cfg.StoragePath("originals")
	if got != "/data/originals" {
		t.Errorf("expected /data/originals, got %s", got)
	}
}

func TestConfig_OriginalsDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.Local.Path = "/data"
	if cfg.OriginalsDir() != "/data/originals" {
		t.Errorf("expected /data/originals, got %s", cfg.OriginalsDir())
	}
}

func TestConfig_ThumbnailsDir(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.Local.Path = "/data"
	if cfg.ThumbnailsDir() != "/data/thumbnails" {
		t.Errorf("expected /data/thumbnails, got %s", cfg.ThumbnailsDir())
	}
}

func TestConfig_MaxFileSize(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Upload.MaxFileSizeMB = 100
	expected := int64(100 * 1024 * 1024)
	if cfg.MaxFileSize() != expected {
		t.Errorf("expected %d, got %d", expected, cfg.MaxFileSize())
	}
}

func TestConfig_IsAllowedExtension_wildcard(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Upload.AllowedExtensions = []string{"*"}
	if !cfg.IsAllowedExtension(".jpg") {
		t.Error("expected wildcard to allow .jpg")
	}
	if !cfg.IsAllowedExtension(".anything") {
		t.Error("expected wildcard to allow .anything")
	}
}

func TestConfig_IsAllowedExtension_empty(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Upload.AllowedExtensions = []string{}
	if !cfg.IsAllowedExtension(".jpg") {
		t.Error("expected empty allow list to allow .jpg")
	}
}

func TestConfig_IsAllowedExtension_specific(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Upload.AllowedExtensions = []string{".jpg", ".png"}
	if !cfg.IsAllowedExtension(".jpg") {
		t.Error("expected .jpg allowed")
	}
	if !cfg.IsAllowedExtension(".JPG") {
		t.Error("expected .JPG allowed (case-insensitive)")
	}
	if cfg.IsAllowedExtension(".pdf") {
		t.Error("expected .pdf denied")
	}
}
