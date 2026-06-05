package store

import (
	"path/filepath"
	"testing"
)

func OpenTestDB(t *testing.T) *DB {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { db.Close() })

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return db
}
