package migrations

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunAppliesSQLiteMigrationsAndCreatesTrackingTable(t *testing.T) {
	db := openSQLiteMemory(t)

	if err := Run(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var name string
	err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='goose_db_version'`,
	).Scan(&name)
	if err != nil {
		t.Fatalf("expected goose_db_version table to exist: %v", err)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	db := openSQLiteMemory(t)

	if err := Run(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	if err := Run(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("second Run should be a no-op: %v", err)
	}
}

func TestRunRejectsUnsupportedDriver(t *testing.T) {
	if err := Run(context.Background(), nil, "mysql"); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

func openSQLiteMemory(t *testing.T) *sql.DB {
	t.Helper()
	// Shared cache so connections in the pool see the same in-memory database.
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
