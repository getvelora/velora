-- +goose Up
-- Reserve schema version 1 as a baseline so future migrations track from a
-- known starting point. The goose_db_version table is created automatically.
SELECT 1;

-- +goose Down
SELECT 1;
