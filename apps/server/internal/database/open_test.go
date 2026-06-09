package database

import (
	"path/filepath"
	"testing"
)

func TestOpenSQLiteEnablesForeignKeys(t *testing.T) {
	db, err := Open(Config{
		Driver: "sqlite",
		URL:    filepath.Join(t.TempDir(), "velora.db"),
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var enabled int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&enabled); err != nil {
		t.Fatalf("query foreign_keys: %v", err)
	}
	if enabled != 1 {
		t.Fatalf("expected foreign keys enabled, got %d", enabled)
	}
}
