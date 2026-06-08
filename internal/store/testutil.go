package store

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"
)

var (
	schemaSnapshotOnce sync.Once
	schemaSnapshot     []string
	schemaSnapshotErr  error
)

var shadowSuffixes = []string{
	"_config", "_content", "_docsize", "_idx", "_data",
	"_node", "_parent", "_rowid",
}

func isShadowTable(name string) bool {
	for _, s := range shadowSuffixes {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

func getSchemaSnapshot() ([]string, error) {
	schemaSnapshotOnce.Do(func() {
		rawDB, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
		if err != nil {
			schemaSnapshotErr = fmt.Errorf("open snapshot db: %w", err)
			return
		}
		defer rawDB.Close()
		rawDB.SetMaxOpenConns(1)

		db := &DB{rawDB}
		if err := db.RunMigrations(); err != nil {
			schemaSnapshotErr = fmt.Errorf("run migrations for snapshot: %w", err)
			return
		}

		rows, err := rawDB.Query(`SELECT name, sql FROM sqlite_master WHERE sql IS NOT NULL
			ORDER BY CASE type WHEN 'table' THEN 0 WHEN 'view' THEN 1 WHEN 'index' THEN 2 WHEN 'trigger' THEN 3 END, name`)
		if err != nil {
			schemaSnapshotErr = fmt.Errorf("query sqlite_master: %w", err)
			return
		}
		defer rows.Close()

		var stmts []string
		for rows.Next() {
			var name, s string
			if err := rows.Scan(&name, &s); err != nil {
				schemaSnapshotErr = fmt.Errorf("scan sqlite_master: %w", err)
				return
			}
			if isShadowTable(name) {
				continue
			}
			stmts = append(stmts, s)
		}
		if err := rows.Err(); err != nil {
			schemaSnapshotErr = fmt.Errorf("rows err: %w", err)
			return
		}
		schemaSnapshot = stmts
	})
	return schemaSnapshot, schemaSnapshotErr
}

func OpenTestDBWithMigrations(t *testing.T) *DB {
	path := t.TempDir() + "/test.db"
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

func OpenTestDB(t *testing.T) *DB {
	path := t.TempDir() + "/test.db"
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	db.SetMaxOpenConns(5)
	t.Cleanup(func() { db.Close() })

	schema, err := getSchemaSnapshot()
	if err != nil {
		t.Fatalf("build schema snapshot: %v", err)
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("apply schema: %v\n%s", err, stmt)
		}
	}

	return db
}
