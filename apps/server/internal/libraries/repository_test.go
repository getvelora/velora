package libraries

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRepositoryCreateAndList(t *testing.T) {
	db := openSQLiteWithSchema(t)
	repo := NewRepository(db, "sqlite")
	ctx := context.Background()

	first, err := repo.Create(ctx, "Movies", "/media/movies")
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if first.ID == 0 {
		t.Fatal("expected non-zero ID on first insert")
	}
	if first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() {
		t.Fatal("expected timestamps to be set on insert")
	}

	second, err := repo.Create(ctx, "TV", "/media/tv")
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}
	if second.ID == first.ID {
		t.Fatalf("expected distinct IDs, both got %d", first.ID)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 libraries, got %d", len(list))
	}
	if list[0].Name != "Movies" || list[1].Name != "TV" {
		t.Fatalf("unexpected list order/contents: %+v", list)
	}
	if list[0].CreatedAt.IsZero() {
		t.Fatal("expected created_at to round-trip from storage")
	}
}

func TestRepositoryCreateRejectsDuplicatePath(t *testing.T) {
	db := openSQLiteWithSchema(t)
	repo := NewRepository(db, "sqlite")
	ctx := context.Background()

	if _, err := repo.Create(ctx, "Movies", "/media/movies"); err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err := repo.Create(ctx, "Movies again", "/media/movies")
	if !errors.Is(err, ErrPathAlreadyExists) {
		t.Fatalf("expected ErrPathAlreadyExists, got %v", err)
	}
}

func TestRepositoryListEmpty(t *testing.T) {
	db := openSQLiteWithSchema(t)
	repo := NewRepository(db, "sqlite")

	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil empty slice for JSON-friendly serialisation")
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d entries", len(list))
	}
}

// openSQLiteWithSchema opens a fresh in-memory SQLite database and applies the
// libraries schema. Mirrors the production migration; updated together when the
// migration changes.
func openSQLiteWithSchema(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared&mode=memory")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(context.Background(), `
		CREATE TABLE libraries (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT    NOT NULL,
			path        TEXT    NOT NULL UNIQUE,
			created_at  TEXT    NOT NULL,
			updated_at  TEXT    NOT NULL
		);
		CREATE INDEX libraries_name_idx ON libraries (name);
	`); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return db
}
