-- +goose Up
ALTER TABLE media_files
    ADD COLUMN inspection_status TEXT NOT NULL DEFAULT 'unprobed'
        CHECK (inspection_status IN ('unprobed', 'ready', 'error')),
    ADD COLUMN inspection_error TEXT,
    ADD COLUMN probed_at TEXT,
    ADD COLUMN format_name TEXT,
    ADD COLUMN format_long_name TEXT,
    ADD COLUMN duration_ms BIGINT,
    ADD COLUMN bit_rate BIGINT;

CREATE TABLE media_streams (
    id                BIGSERIAL PRIMARY KEY,
    media_file_id     BIGINT  NOT NULL REFERENCES media_files (id) ON DELETE CASCADE,
    stream_index      INTEGER NOT NULL,
    stream_type       TEXT    NOT NULL CHECK (stream_type IN ('video', 'audio', 'subtitle')),
    codec_name        TEXT    NOT NULL DEFAULT '',
    codec_long_name   TEXT    NOT NULL DEFAULT '',
    profile           TEXT    NOT NULL DEFAULT '',
    level             INTEGER,
    language          TEXT    NOT NULL DEFAULT '',
    title             TEXT    NOT NULL DEFAULT '',
    is_default        BOOLEAN NOT NULL DEFAULT FALSE,
    is_forced         BOOLEAN NOT NULL DEFAULT FALSE,
    width             INTEGER NOT NULL DEFAULT 0,
    height            INTEGER NOT NULL DEFAULT 0,
    pixel_format      TEXT    NOT NULL DEFAULT '',
    bit_depth         INTEGER NOT NULL DEFAULT 0,
    frame_rate        TEXT    NOT NULL DEFAULT '',
    color_range       TEXT    NOT NULL DEFAULT '',
    color_space       TEXT    NOT NULL DEFAULT '',
    color_transfer    TEXT    NOT NULL DEFAULT '',
    color_primaries   TEXT    NOT NULL DEFAULT '',
    sample_rate       INTEGER NOT NULL DEFAULT 0,
    channels          INTEGER NOT NULL DEFAULT 0,
    channel_layout    TEXT    NOT NULL DEFAULT '',
    UNIQUE (media_file_id, stream_index)
);

CREATE INDEX media_streams_file_idx ON media_streams (media_file_id);

-- +goose Down
DROP INDEX IF EXISTS media_streams_file_idx;
DROP TABLE IF EXISTS media_streams;
ALTER TABLE media_files
    DROP COLUMN bit_rate,
    DROP COLUMN duration_ms,
    DROP COLUMN format_long_name,
    DROP COLUMN format_name,
    DROP COLUMN probed_at,
    DROP COLUMN inspection_error,
    DROP COLUMN inspection_status;
