package store

import (
	"testing"
)

func OpenTestDB(t *testing.T) *DB {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return db
}
