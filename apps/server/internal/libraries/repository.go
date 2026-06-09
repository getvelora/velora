package libraries

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrPathAlreadyExists is returned by Repository.Create when the path uniqueness
// constraint is violated. Handlers translate this into a 409 Conflict.
var (
	ErrPathAlreadyExists = errors.New("library with this path already exists")
	ErrNotFound          = errors.New("library not found")
)

// Repository persists Library values. It supports both SQLite and Postgres
// via the driver name passed at construction time; timestamps are stored as
// RFC3339Nano text in both dialects to sidestep driver-specific time handling.
type Repository struct {
	db     *sql.DB
	driver string
}

// NewRepository binds a Repository to an open *sql.DB. driver must be the
// VELORA_DATABASE_DRIVER value (sqlite or postgres) so the INSERT path can pick
// between LastInsertId (sqlite) and RETURNING id (postgres).
func NewRepository(db *sql.DB, driver string) *Repository {
	return &Repository{db: db, driver: driver}
}

// List returns every library, ordered by id (i.e. insertion order).
func (r *Repository) List(ctx context.Context) ([]Library, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, path, created_at, updated_at
		FROM libraries
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("query libraries: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	out := []Library{}
	for rows.Next() {
		var lib Library
		var created, updated string
		if err := rows.Scan(&lib.ID, &lib.Name, &lib.Path, &created, &updated); err != nil {
			return nil, fmt.Errorf("scan library: %w", err)
		}
		if lib.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return nil, fmt.Errorf("parse created_at %q: %w", created, err)
		}
		if lib.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
			return nil, fmt.Errorf("parse updated_at %q: %w", updated, err)
		}
		out = append(out, lib)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (Library, error) {
	query := `
		SELECT id, name, path, created_at, updated_at
		FROM libraries
		WHERE id = ?
	`
	if r.driver == "postgres" {
		query = `
			SELECT id, name, path, created_at, updated_at
			FROM libraries
			WHERE id = $1
		`
	}

	var library Library
	var created, updated string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&library.ID,
		&library.Name,
		&library.Path,
		&created,
		&updated,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Library{}, ErrNotFound
	}
	if err != nil {
		return Library{}, fmt.Errorf("query library %d: %w", id, err)
	}
	if library.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Library{}, fmt.Errorf("parse created_at %q: %w", created, err)
	}
	if library.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return Library{}, fmt.Errorf("parse updated_at %q: %w", updated, err)
	}
	return library, nil
}

// Create inserts a new library with the given name and path. Returns
// ErrPathAlreadyExists if the path is already taken.
func (r *Repository) Create(ctx context.Context, name, path string) (Library, error) {
	now := time.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)

	switch r.driver {
	case "postgres":
		var id int64
		err := r.db.QueryRowContext(ctx, `
			INSERT INTO libraries (name, path, created_at, updated_at)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, name, path, nowText, nowText).Scan(&id)
		if err != nil {
			if isUniqueViolation(err) {
				return Library{}, ErrPathAlreadyExists
			}
			return Library{}, fmt.Errorf("insert library: %w", err)
		}
		return Library{ID: id, Name: name, Path: path, CreatedAt: now, UpdatedAt: now}, nil

	default: // sqlite
		res, err := r.db.ExecContext(ctx, `
			INSERT INTO libraries (name, path, created_at, updated_at)
			VALUES (?, ?, ?, ?)
		`, name, path, nowText, nowText)
		if err != nil {
			if isUniqueViolation(err) {
				return Library{}, ErrPathAlreadyExists
			}
			return Library{}, fmt.Errorf("insert library: %w", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return Library{}, fmt.Errorf("get insert id: %w", err)
		}
		return Library{ID: id, Name: name, Path: path, CreatedAt: now, UpdatedAt: now}, nil
	}
}

// isUniqueViolation maps the driver-specific unique-constraint error strings
// onto a single boolean. modernc.org/sqlite and pgx don't share a unified
// error code, so we match on substrings rather than depend on either's
// typed-error API.
func isUniqueViolation(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}
