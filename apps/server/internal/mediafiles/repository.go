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
			first_seen_at, last_seen_at, missing_at, inspection_status,
			inspection_error, probed_at, format_name, format_long_name,
			duration_ms, bit_rate
		FROM media_files
		WHERE library_id = ?
		ORDER BY path
	`
	if r.driver == "postgres" {
		query = `
			SELECT id, library_id, path, size, modified_at, status,
				first_seen_at, last_seen_at, missing_at, inspection_status,
				inspection_error, probed_at, format_name, format_long_name,
				duration_ms, bit_rate
			FROM media_files
			WHERE library_id = $1
			ORDER BY path
		`
	}

	rows, err := r.db.QueryContext(ctx, query, libraryID)
	if err != nil {
		return nil, fmt.Errorf("query media files: %w", err)
	}
	files := []File{}
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate media files: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close media files: %w", err)
	}
	if err := r.loadStreams(ctx, libraryID, files); err != nil {
		return nil, err
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
			var id int64
			if id, err = r.insert(ctx, tx, libraryID, item, scannedAt); err != nil {
				return summary, err
			}
			if item.Inspection.Attempted {
				if err = r.saveInspection(ctx, tx, id, item.Inspection, scannedAt); err != nil {
					return summary, err
				}
			}
			summary.Added++
		case current.Status == StatusMissing:
			if err = r.updateAvailable(ctx, tx, current.ID, item, scannedAt); err != nil {
				return summary, err
			}
			if item.Inspection.Attempted {
				if err = r.saveInspection(ctx, tx, current.ID, item.Inspection, scannedAt); err != nil {
					return summary, err
				}
			}
			summary.Restored++
		case current.Size != item.Size || !current.ModifiedAt.Equal(item.ModifiedAt):
			if err = r.updateAvailable(ctx, tx, current.ID, item, scannedAt); err != nil {
				return summary, err
			}
			if item.Inspection.Attempted {
				if err = r.saveInspection(ctx, tx, current.ID, item.Inspection, scannedAt); err != nil {
					return summary, err
				}
			}
			summary.Updated++
		default:
			if err = r.touch(ctx, tx, current.ID, scannedAt); err != nil {
				return summary, err
			}
			if item.Inspection.Attempted {
				if err = r.saveInspection(ctx, tx, current.ID, item.Inspection, scannedAt); err != nil {
					return summary, err
				}
			}
			summary.Unchanged++
		}
		if item.Inspection.Attempted {
			summary.Probed++
			if item.Inspection.Error != "" {
				summary.ProbeFailed++
			}
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
			first_seen_at, last_seen_at, missing_at, inspection_status,
			inspection_error, probed_at, format_name, format_long_name,
			duration_ms, bit_rate
		FROM media_files
		WHERE library_id = ?
	`
	if r.driver == "postgres" {
		query = `
			SELECT id, library_id, path, size, modified_at, status,
				first_seen_at, last_seen_at, missing_at, inspection_status,
				inspection_error, probed_at, format_name, format_long_name,
				duration_ms, bit_rate
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
	var missingAt, inspectionError, probedAt sql.NullString
	var formatName, formatLongName sql.NullString
	var durationMS, bitRate sql.NullInt64
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
		&file.Inspection.Status,
		&inspectionError,
		&probedAt,
		&formatName,
		&formatLongName,
		&durationMS,
		&bitRate,
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
	if inspectionError.Valid {
		file.Inspection.Error = &inspectionError.String
	}
	if probedAt.Valid {
		value, err := parseTime("probed_at", probedAt.String)
		if err != nil {
			return File{}, err
		}
		file.Inspection.ProbedAt = &value
	}
	if file.Inspection.Status == InspectionStatusReady {
		file.Inspection.Format = &Format{
			Name:       formatName.String,
			LongName:   formatLongName.String,
			DurationMS: durationMS.Int64,
			BitRate:    bitRate.Int64,
		}
	}
	file.Inspection.Streams = []Stream{}
	return file, nil
}

func parseTime(field, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s %q: %w", field, value, err)
	}
	return parsed, nil
}

func (r *Repository) loadStreams(ctx context.Context, libraryID int64, files []File) error {
	if len(files) == 0 {
		return nil
	}
	query := `
		SELECT s.media_file_id, s.stream_index, s.stream_type, s.codec_name,
			s.codec_long_name, s.profile, s.level, s.language, s.title,
			s.is_default, s.is_forced, s.width, s.height, s.pixel_format,
			s.bit_depth, s.frame_rate, s.color_range, s.color_space,
			s.color_transfer, s.color_primaries, s.sample_rate, s.channels,
			s.channel_layout
		FROM media_streams s
		JOIN media_files f ON f.id = s.media_file_id
		WHERE f.library_id = ?
		ORDER BY s.media_file_id, s.stream_index
	`
	if r.driver == "postgres" {
		query = `
			SELECT s.media_file_id, s.stream_index, s.stream_type, s.codec_name,
				s.codec_long_name, s.profile, s.level, s.language, s.title,
				s.is_default, s.is_forced, s.width, s.height, s.pixel_format,
				s.bit_depth, s.frame_rate, s.color_range, s.color_space,
				s.color_transfer, s.color_primaries, s.sample_rate, s.channels,
				s.channel_layout
			FROM media_streams s
			JOIN media_files f ON f.id = s.media_file_id
			WHERE f.library_id = $1
			ORDER BY s.media_file_id, s.stream_index
		`
	}
	rows, err := r.db.QueryContext(ctx, query, libraryID)
	if err != nil {
		return fmt.Errorf("query media streams: %w", err)
	}
	defer func() { _ = rows.Close() }()

	fileIndexes := make(map[int64]int, len(files))
	for index := range files {
		fileIndexes[files[index].ID] = index
	}
	for rows.Next() {
		var mediaFileID int64
		var stream Stream
		var level sql.NullInt64
		if err := rows.Scan(
			&mediaFileID,
			&stream.Index,
			&stream.Type,
			&stream.CodecName,
			&stream.CodecLongName,
			&stream.Profile,
			&level,
			&stream.Language,
			&stream.Title,
			&stream.Default,
			&stream.Forced,
			&stream.Width,
			&stream.Height,
			&stream.PixelFormat,
			&stream.BitDepth,
			&stream.FrameRate,
			&stream.ColorRange,
			&stream.ColorSpace,
			&stream.ColorTransfer,
			&stream.ColorPrimaries,
			&stream.SampleRate,
			&stream.Channels,
			&stream.ChannelLayout,
		); err != nil {
			return fmt.Errorf("scan media stream: %w", err)
		}
		if level.Valid {
			value := int(level.Int64)
			stream.Level = &value
		}
		if index, ok := fileIndexes[mediaFileID]; ok {
			files[index].Inspection.Streams = append(files[index].Inspection.Streams, stream)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate media streams: %w", err)
	}
	return nil
}

func (r *Repository) insert(
	ctx context.Context,
	tx *sql.Tx,
	libraryID int64,
	item DiscoveredFile,
	scannedAt time.Time,
) (int64, error) {
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
			RETURNING id
		`
		var id int64
		if err := tx.QueryRowContext(
			ctx, query, libraryID, item.Path, item.Size, formatTime(item.ModifiedAt),
			StatusAvailable, formatTime(scannedAt), formatTime(scannedAt),
		).Scan(&id); err != nil {
			return 0, fmt.Errorf("insert media file %q: %w", item.Path, err)
		}
		return id, nil
	}
	result, err := tx.ExecContext(
		ctx,
		query,
		libraryID,
		item.Path,
		item.Size,
		formatTime(item.ModifiedAt),
		StatusAvailable,
		formatTime(scannedAt),
		formatTime(scannedAt),
	)
	if err != nil {
		return 0, fmt.Errorf("insert media file %q: %w", item.Path, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("get media file insert id: %w", err)
	}
	return id, nil
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

func (r *Repository) saveInspection(
	ctx context.Context,
	tx *sql.Tx,
	mediaFileID int64,
	inspection InspectionResult,
	probedAt time.Time,
) error {
	status := InspectionStatusReady
	var errorValue any
	var formatName, formatLongName any = inspection.Result.Format.Name, inspection.Result.Format.LongName
	var durationMS, bitRate any = inspection.Result.Format.DurationMS, inspection.Result.Format.BitRate
	if inspection.Error != "" {
		status = InspectionStatusError
		errorValue = inspection.Error
		formatName = nil
		formatLongName = nil
		durationMS = nil
		bitRate = nil
	}

	query := `
		UPDATE media_files
		SET inspection_status = ?, inspection_error = ?, probed_at = ?,
			format_name = ?, format_long_name = ?, duration_ms = ?, bit_rate = ?
		WHERE id = ?
	`
	if r.driver == "postgres" {
		query = `
			UPDATE media_files
			SET inspection_status = $1, inspection_error = $2, probed_at = $3,
				format_name = $4, format_long_name = $5, duration_ms = $6, bit_rate = $7
			WHERE id = $8
		`
	}
	if _, err := tx.ExecContext(
		ctx,
		query,
		status,
		errorValue,
		formatTime(probedAt),
		formatName,
		formatLongName,
		durationMS,
		bitRate,
		mediaFileID,
	); err != nil {
		return fmt.Errorf("update inspection for media file %d: %w", mediaFileID, err)
	}

	deleteQuery := `DELETE FROM media_streams WHERE media_file_id = ?`
	if r.driver == "postgres" {
		deleteQuery = `DELETE FROM media_streams WHERE media_file_id = $1`
	}
	if _, err := tx.ExecContext(ctx, deleteQuery, mediaFileID); err != nil {
		return fmt.Errorf("delete streams for media file %d: %w", mediaFileID, err)
	}
	if status == InspectionStatusError {
		return nil
	}

	for _, stream := range inspection.Result.Streams {
		if err := r.insertStream(ctx, tx, mediaFileID, stream); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) insertStream(
	ctx context.Context,
	tx *sql.Tx,
	mediaFileID int64,
	stream Stream,
) error {
	query := `
		INSERT INTO media_streams (
			media_file_id, stream_index, stream_type, codec_name, codec_long_name,
			profile, level, language, title, is_default, is_forced, width, height,
			pixel_format, bit_depth, frame_rate, color_range, color_space,
			color_transfer, color_primaries, sample_rate, channels, channel_layout
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	if r.driver == "postgres" {
		query = `
			INSERT INTO media_streams (
				media_file_id, stream_index, stream_type, codec_name, codec_long_name,
				profile, level, language, title, is_default, is_forced, width, height,
				pixel_format, bit_depth, frame_rate, color_range, color_space,
				color_transfer, color_primaries, sample_rate, channels, channel_layout
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
				$13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
			)
		`
	}
	if _, err := tx.ExecContext(
		ctx,
		query,
		mediaFileID,
		stream.Index,
		stream.Type,
		stream.CodecName,
		stream.CodecLongName,
		stream.Profile,
		stream.Level,
		stream.Language,
		stream.Title,
		stream.Default,
		stream.Forced,
		stream.Width,
		stream.Height,
		stream.PixelFormat,
		stream.BitDepth,
		stream.FrameRate,
		stream.ColorRange,
		stream.ColorSpace,
		stream.ColorTransfer,
		stream.ColorPrimaries,
		stream.SampleRate,
		stream.Channels,
		stream.ChannelLayout,
	); err != nil {
		return fmt.Errorf(
			"insert stream %d for media file %d: %w",
			stream.Index,
			mediaFileID,
			err,
		)
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
