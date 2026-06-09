-- +goose Up
CREATE TABLE media_files (
    id             BIGSERIAL PRIMARY KEY,
    library_id     BIGINT NOT NULL REFERENCES libraries (id) ON DELETE CASCADE,
    path           TEXT   NOT NULL,
    size           BIGINT NOT NULL,
    modified_at    TEXT   NOT NULL,
    status         TEXT   NOT NULL CHECK (status IN ('available', 'missing')),
    first_seen_at  TEXT   NOT NULL,
    last_seen_at   TEXT   NOT NULL,
    missing_at     TEXT,
    UNIQUE (library_id, path)
);

CREATE INDEX media_files_library_status_idx ON media_files (library_id, status);

-- +goose Down
DROP INDEX IF EXISTS media_files_library_status_idx;
DROP TABLE IF EXISTS media_files;
