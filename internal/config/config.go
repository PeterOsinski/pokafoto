package config

import (
	"os"
	"strconv"
	"strings"
)

type S3Config struct {
	Enabled    bool   `yaml:"enabled"`
	Endpoint   string `yaml:"endpoint"`
	Bucket     string `yaml:"bucket"`
	AccessKey  string `yaml:"access_key"`
	SecretKey  string `yaml:"secret_key"`
	Region     string `yaml:"region"`
	UseSSL     bool   `yaml:"use_ssl"`
}

type LocalStorageConfig struct {
	Path string `yaml:"path"`
}

type StorageConfig struct {
	S3               S3Config          `yaml:"s3"`
	Local            LocalStorageConfig `yaml:"local"`
	MaxDiskUsagePct  int                `yaml:"max_disk_usage_pct"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	AllowRegistration         bool   `yaml:"allow_registration"`
	JWTSecret                 string `yaml:"jwt_secret"`
	SessionDurationH          int    `yaml:"session_duration_hours"`
	FolderPasswordExpiryMinutes int  `yaml:"folder_password_expiry_minutes"`
}

type ThumbnailSizeConfig struct {
	Width       int    `yaml:"width"`
	Quality     int    `yaml:"quality"`
	Format      string `yaml:"format"`
	MaxDimension int   `yaml:"max_dimension,omitempty"`
}

type MediaConfig struct {
	AutoOrganize        bool                 `yaml:"auto_organize"`
	OrganizationPattern string               `yaml:"organization_pattern"`
	ThumbnailSizes      map[string]ThumbnailSizeConfig `yaml:"thumbnail_sizes"`
}

type UploadConfig struct {
	MaxFileSizeMB         int64    `yaml:"max_file_size_mb"`
	ConcurrentWorkers     int      `yaml:"concurrent_workers"`
	AllowedExtensions     []string `yaml:"allowed_extensions"`
	ChunkSizeMB           int      `yaml:"chunk_size_mb"`
	ChunkCleanupHours     int      `yaml:"chunk_cleanup_hours"`
	MaxChunkUploadAgeHours int     `yaml:"max_chunk_upload_age_hours"`
	ChunkThresholdMB      int      `yaml:"chunk_threshold_mb"`
}

type MapConfig struct {
	TileSource       string `yaml:"tile_source"`
	MaxClusterRadius int    `yaml:"max_cluster_radius"`
}

type BackupConfig struct {
	Enabled       bool `yaml:"enabled"`
	IntervalH     int  `yaml:"interval_h"`
	RetentionDays int  `yaml:"retention_days"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Config struct {
	Server              ServerConfig   `yaml:"server"`
	Storage             StorageConfig  `yaml:"storage"`
	Database            DatabaseConfig `yaml:"database"`
	Auth                AuthConfig     `yaml:"auth"`
	Media               MediaConfig    `yaml:"media"`
	Upload              UploadConfig   `yaml:"upload"`
	Map                 MapConfig      `yaml:"map"`
	TrashExpirationDays int            `yaml:"trash_expiration_days"`
	Backup              BackupConfig   `yaml:"backup"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Storage: StorageConfig{
			S3: S3Config{
				Enabled:    false,
				Endpoint:   "",
				Bucket:     "",
				AccessKey:  "",
				SecretKey:  "",
				Region:     "",
				UseSSL:     true,
			},
			Local: LocalStorageConfig{
				Path: "./data",
			},
			MaxDiskUsagePct: 70,
		},
		Database: DatabaseConfig{
			Path: "./data/drive.db",
		},
		Auth: AuthConfig{
			AllowRegistration:           false,
			JWTSecret:                   "",
			SessionDurationH:            72,
			FolderPasswordExpiryMinutes: 30,
		},
		Media: MediaConfig{
			AutoOrganize:        true,
			OrganizationPattern: "{{year}}/{{month}}",
			ThumbnailSizes: map[string]ThumbnailSizeConfig{
				"small":       {Width: 60, Quality: 60, Format: "jpeg"},
				"large":       {Width: 300, Quality: 75, Format: "jpeg"},
				"medium":      {Width: 600, Quality: 75, Format: "jpeg"},
				"xl":          {Width: 2000, Quality: 85, Format: "jpeg"},
				"preview":     {MaxDimension: 720, Quality: 80, Format: "webp"},
				"video_still": {MaxDimension: 600, Quality: 75, Format: "jpeg"},
			},
		},
		Upload: UploadConfig{
			MaxFileSizeMB:         10240,
			ConcurrentWorkers:     4,
			AllowedExtensions:     []string{"*"},
			ChunkSizeMB:           5,
			ChunkCleanupHours:     24,
			MaxChunkUploadAgeHours: 48,
			ChunkThresholdMB:      50,
		},
		Map: MapConfig{
			TileSource:       "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
			MaxClusterRadius: 80,
		},
		TrashExpirationDays: 30,
		Backup: BackupConfig{
			Enabled:       false,
			IntervalH:     24,
			RetentionDays: 0,
		},
	}
}

func Load() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("DRIVE_STORAGE_PATH"); v != "" {
		cfg.Storage.Local.Path = v
	}
	if v := os.Getenv("DRIVE_DB_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("DRIVE_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("DRIVE_ALLOW_REGISTRATION"); v != "" {
		cfg.Auth.AllowRegistration = v == "true"
	}
	if v := os.Getenv("DRIVE_S3_ENABLED"); v != "" {
		cfg.Storage.S3.Enabled = v == "true"
	}
	if v := os.Getenv("DRIVE_S3_ENDPOINT"); v != "" {
		cfg.Storage.S3.Endpoint = v
	}
	if v := os.Getenv("DRIVE_S3_BUCKET"); v != "" {
		cfg.Storage.S3.Bucket = v
	}
	if v := os.Getenv("DRIVE_S3_ACCESS_KEY"); v != "" {
		cfg.Storage.S3.AccessKey = v
	}
	if v := os.Getenv("DRIVE_S3_SECRET_KEY"); v != "" {
		cfg.Storage.S3.SecretKey = v
	}
	if v := os.Getenv("DRIVE_S3_REGION"); v != "" {
		cfg.Storage.S3.Region = v
	}
	if v := os.Getenv("DRIVE_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("DRIVE_MAX_DISK_USAGE_PCT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 && p <= 100 {
			cfg.Storage.MaxDiskUsagePct = p
		}
	}
	if v := os.Getenv("DRIVE_TRASH_EXPIRATION_DAYS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 {
			cfg.TrashExpirationDays = d
		}
	}
	if v := os.Getenv("DRIVE_BACKUP_ENABLED"); v != "" {
		cfg.Backup.Enabled = v == "true"
	}
	if v := os.Getenv("DRIVE_BACKUP_INTERVAL_H"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 {
			cfg.Backup.IntervalH = h
		}
	}
	if v := os.Getenv("DRIVE_BACKUP_RETENTION_DAYS"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d >= 0 {
			cfg.Backup.RetentionDays = d
		}
	}

	return cfg
}

func (c *Config) StoragePath(subdirs ...string) string {
	p := c.Storage.Local.Path
	for _, d := range subdirs {
		p = p + "/" + d
	}
	return p
}

func (c *Config) OriginalsDir() string {
	return c.StoragePath("originals")
}

func (c *Config) ThumbnailsDir() string {
	return c.StoragePath("thumbnails")
}

func (c *Config) WebDistPath() string {
	return c.StoragePath("../web/dist")
}

func (c *Config) MaxFileSize() int64 {
	return c.Upload.MaxFileSizeMB * 1024 * 1024
}

func (c *Config) IsAllowedExtension(ext string) bool {
	if len(c.Upload.AllowedExtensions) == 0 {
		return true
	}
	if c.Upload.AllowedExtensions[0] == "*" {
		return true
	}
	ext = strings.ToLower(ext)
	for _, allowed := range c.Upload.AllowedExtensions {
		if strings.ToLower(allowed) == ext {
			return true
		}
	}
	return false
}

func (c *Config) MaxDiskUsagePercent() int {
	if c.Storage.MaxDiskUsagePct > 0 && c.Storage.MaxDiskUsagePct <= 100 {
		return c.Storage.MaxDiskUsagePct
	}
	return 70
}
