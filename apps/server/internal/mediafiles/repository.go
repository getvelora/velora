package mediafiles

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Repository struct {
	db     *sql.DB
	driver string
}

func NewRepository(db *sql.DB, driver string) *Repository {
	return &Repository{db: db, driver: driver}
}

func (r *Repository) List(ctx context.Context, libraryID int64) ([]File, error) {
	query := `
		SELECT id, library_id, path, size, modified_at, status,
			first_seen_at, last_seen_at, missing_at
		FROM media_files
		WHERE library_id = ?
		ORDER BY path
	`
	if r.driver == "postgres" {
		query = `
			SELECT id, library_id, path, size, modified_at, status,
				first_seen_at, last_seen_at, missing_at
			FROM media_files
			WHERE library_id = $1
			ORDER BY path
		`
	}

	rows, err := r.db.QueryContext(ctx, query, libraryID)
	if err != nil {
		return nil, fmt.Errorf("query media files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	files := []File{}
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate media files: %w", err)
	}
	return files, nil
}

func (r *Repository) Reconcile(
	ctx context.Context,
	libraryID int64,
	discovered []DiscoveredFile,
	scannedAt time.Time,
) (summary ReconcileSummary, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return summary, fmt.Errorf("begin media file reconciliation: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	existing, err := r.listForUpdate(ctx, tx, libraryID)
	if err != nil {
		return summary, err
	}

	summary.Discovered = len(discovered)
	seen := make(map[string]struct{}, len(discovered))
	for _, item := range discovered {
		seen[item.Path] = struct{}{}
		current, found := existing[item.Path]
		switch {
		case !found:
			if err = r.insert(ctx, tx, libraryID, item, scannedAt); err != nil {
				return summary, err
			}
			summary.Added++
		case current.Status == StatusMissing:
			if err = r.updateAvailable(ctx, tx, current.ID, item, scannedAt); err != nil {
				return summary, err
			}
			summary.Restored++
		case current.Size != item.Size || !current.ModifiedAt.Equal(item.ModifiedAt):
			if err = r.updateAvailable(ctx, tx, current.ID, item, scannedAt); err != nil {
				return summary, err
			}
			summary.Updated++
		default:
			if err = r.touch(ctx, tx, current.ID, scannedAt); err != nil {
				return summary, err
			}
			summary.Unchanged++
		}
	}

	for path, current := range existing {
		if _, found := seen[path]; found || current.Status == StatusMissing {
			continue
		}
		if err = r.markMissing(ctx, tx, current.ID, scannedAt); err != nil {
			return summary, err
		}
		summary.MarkedMissing++
	}

	if err = tx.Commit(); err != nil {
		return summary, fmt.Errorf("commit media file reconciliation: %w", err)
	}
	return summary, nil
}

func (r *Repository) listForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
) (map[string]File, error) {
	query := `
		SELECT id, library_id, path, size, modified_at, status,
			first_seen_at, last_seen_at, missing_at
		FROM media_files
		WHERE library_id = ?
	`
	if r.driver == "postgres" {
		query = `
			SELECT id, library_id, path, size, modified_at, status,
				first_seen_at, last_seen_at, missing_at
			FROM media_files
			WHERE library_id = $1
			FOR UPDATE
		`
	}
	rows, err := tx.QueryContext(ctx, query, libraryID)
	if err != nil {
		return nil, fmt.Errorf("query existing media files: %w", err)
	}
	defer func() { _ = rows.Close() }()

	files := map[string]File{}
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files[file.Path] = file
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing media files: %w", err)
	}
	return files, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFile(row rowScanner) (File, error) {
	var file File
	var modifiedAt, firstSeenAt, lastSeenAt string
	var missingAt sql.NullString
	if err := row.Scan(
		&file.ID,
		&file.LibraryID,
		&file.Path,
		&file.Size,
		&modifiedAt,
		&file.Status,
		&firstSeenAt,
		&lastSeenAt,
		&missingAt,
	); err != nil {
		return File{}, fmt.Errorf("scan media file: %w", err)
	}

	var err error
	if file.ModifiedAt, err = parseTime("modified_at", modifiedAt); err != nil {
		return File{}, err
	}
	if file.FirstSeenAt, err = parseTime("first_seen_at", firstSeenAt); err != nil {
		return File{}, err
	}
	if file.LastSeenAt, err = parseTime("last_seen_at", lastSeenAt); err != nil {
		return File{}, err
	}
	if missingAt.Valid {
		value, err := parseTime("missing_at", missingAt.String)
		if err != nil {
			return File{}, err
		}
		file.MissingAt = &value
	}
	return file, nil
}

func parseTime(field, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s %q: %w", field, value, err)
	}
	return parsed, nil
}

func (r *Repository) insert(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	item DiscoveredFile,
	scannedAt time.Time,
) error {
	query := `
		INSERT INTO media_files (
			library_id, path, size, modified_at, status, first_seen_at, last_seen_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	if r.driver == "postgres" {
		query = `
			INSERT INTO media_files (
				library_id, path, size, modified_at, status, first_seen_at, last_seen_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
	}
	if _, err := tx.ExecContext(
		ctx,
		query,
		libraryID,
		item.Path,
		item.Size,
		formatTime(item.ModifiedAt),
		StatusAvailable,
		formatTime(scannedAt),
		formatTime(scannedAt),
	); err != nil {
		return fmt.Errorf("insert media file %q: %w", item.Path, err)
	}
	return nil
}

func (r *Repository) updateAvailable(
	ctx context.Context,
	tx *sql.Tx,
	id int64,
	item DiscoveredFile,
	scannedAt time.Time,
) error {
	query := `
		UPDATE media_files
		SET size = ?, modified_at = ?, status = ?, last_seen_at = ?, missing_at = NULL
		WHERE id = ?
	`
	if r.driver == "postgres" {
		query = `
			UPDATE media_files
			SET size = $1, modified_at = $2, status = $3, last_seen_at = $4, missing_at = NULL
			WHERE id = $5
		`
	}
	if _, err := tx.ExecContext(
		ctx,
		query,
		item.Size,
		formatTime(item.ModifiedAt),
		StatusAvailable,
		formatTime(scannedAt),
		id,
	); err != nil {
		return fmt.Errorf("update media file %q: %w", item.Path, err)
	}
	return nil
}

func (r *Repository) touch(
	ctx context.Context,
	tx *sql.Tx,
	id int64,
	scannedAt time.Time,
) error {
	query := `UPDATE media_files SET last_seen_at = ? WHERE id = ?`
	if r.driver == "postgres" {
		query = `UPDATE media_files SET last_seen_at = $1 WHERE id = $2`
	}
	if _, err := tx.ExecContext(ctx, query, formatTime(scannedAt), id); err != nil {
		return fmt.Errorf("touch media file %d: %w", id, err)
	}
	return nil
}

func (r *Repository) markMissing(
	ctx context.Context,
	tx *sql.Tx,
	id int64,
	scannedAt time.Time,
) error {
	query := `UPDATE media_files SET status = ?, missing_at = ? WHERE id = ?`
	if r.driver == "postgres" {
		query = `UPDATE media_files SET status = $1, missing_at = $2 WHERE id = $3`
	}
	if _, err := tx.ExecContext(
		ctx,
		query,
		StatusMissing,
		formatTime(scannedAt),
		id,
	); err != nil {
		return fmt.Errorf("mark media file %d missing: %w", id, err)
	}
	return nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
