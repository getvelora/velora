-- +goose Up
CREATE TABLE libraries (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT      NOT NULL,
    path        TEXT      NOT NULL UNIQUE,
    created_at  TEXT      NOT NULL,
    updated_at  TEXT      NOT NULL
);

CREATE INDEX libraries_name_idx ON libraries (name);

-- +goose Down
DROP INDEX IF EXISTS libraries_name_idx;
DROP TABLE IF EXISTS libraries;
