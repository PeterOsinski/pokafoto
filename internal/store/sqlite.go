package store

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migration_*.sql
var migrationsFS embed.FS

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) RunMigrations() error {
	entries, err := migrationsFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var migrationFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			migrationFiles = append(migrationFiles, e.Name())
		}
	}
	sort.Strings(migrationFiles)

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}

	for _, m := range migrationFiles {
		var applied int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", versionFromName(m)).Scan(&applied)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", m, err)
		}
		if applied > 0 {
			slog.Debug("migration already applied", "migration", m)
			continue
		}

		content, err := migrationsFS.ReadFile(m)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration tx: %w", err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", m, err)
		}

		version := versionFromName(m)
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m, err)
		}

		slog.Info("applied migration", "migration", m, "version", version)
	}

	return nil
}

func versionFromName(name string) int {
	name = strings.TrimPrefix(name, "migration_")
	var v int
	fmt.Sscanf(name, "%d", &v)
	return v
}
