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

	err = db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='media_files'`,
	).Scan(&name)
	if err != nil {
		t.Fatalf("expected media_files table to exist: %v", err)
	}
}

func TestSQLiteMediaFilesCascadeWhenLibraryIsDeleted(t *testing.T) {
	db := openSQLiteMemory(t)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := Run(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO libraries (id, name, path, created_at, updated_at)
		VALUES (1, 'Movies', '/media/movies', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO media_files (
			library_id, path, size, modified_at, status, first_seen_at, last_seen_at
		) VALUES (
			1, 'movie.mkv', 100, '2026-01-01T00:00:00Z', 'available',
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		);
		DELETE FROM libraries WHERE id = 1;
	`); err != nil {
		t.Fatalf("insert and delete library: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media_files`).Scan(&count); err != nil {
		t.Fatalf("count media files: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cascade delete, got %d media files", count)
	}
}

func TestSQLiteInspectionMigrationDefaultsExistingFilesAndCascadesStreams(t *testing.T) {
	db := openSQLiteMemory(t)
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if err := Run(context.Background(), db, "sqlite"); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO libraries (id, name, path, created_at, updated_at)
		VALUES (1, 'Movies', '/media/movies', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
		INSERT INTO media_files (
			id, library_id, path, size, modified_at, status, first_seen_at, last_seen_at
		) VALUES (
			10, 1, 'movie.mkv', 100, '2026-01-01T00:00:00Z', 'available',
			'2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'
		);
	`); err != nil {
		t.Fatalf("insert media file: %v", err)
	}

	var status string
	if err := db.QueryRow(
		`SELECT inspection_status FROM media_files WHERE id = 10`,
	).Scan(&status); err != nil {
		t.Fatalf("query inspection status: %v", err)
	}
	if status != "unprobed" {
		t.Fatalf("expected unprobed, got %q", status)
	}

	if _, err := db.Exec(`
		INSERT INTO media_streams (media_file_id, stream_index, stream_type, codec_name)
		VALUES (10, 0, 'video', 'h264');
		DELETE FROM media_files WHERE id = 10;
	`); err != nil {
		t.Fatalf("insert stream and delete media file: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM media_streams`).Scan(&count); err != nil {
		t.Fatalf("count media streams: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected cascade delete, got %d streams", count)
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
