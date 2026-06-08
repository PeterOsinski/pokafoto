package store

import (
	"testing"
)

func TestSqlite_Open_shouldConnect(t *testing.T) {
	db := OpenTestDB(t)
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestSqlite_RunMigrations_shouldApplyAll(t *testing.T) {
	db := OpenTestDBWithMigrations(t)

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count < 2 {
		t.Errorf("expected at least 2 migrations, got %d", count)
	}
}

func TestSqlite_RunMigrations_shouldBeIdempotent(t *testing.T) {
	db := OpenTestDBWithMigrations(t)

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("second migration run: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if count < 2 {
		t.Errorf("expected migrations still applied, got %d", count)
	}
}

func TestSqlite_VersionFromName_shouldParseNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"from 001", "migration_001_initial.sql", 1},
		{"from 002", "migration_002_fts5.sql", 2},
		{"from 999", "migration_999_something.sql", 999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionFromName(tt.input)
			if got != tt.expected {
				t.Errorf("versionFromName(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSqlite_TablesExist_afterMigrations(t *testing.T) {
	db := OpenTestDB(t)

	tables := []string{"users", "sessions", "files", "exif", "thumbnails", "geo_index_meta"}
	for _, table := range tables {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count); err != nil {
			t.Errorf("check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("table %s does not exist", table)
		}
	}
}
