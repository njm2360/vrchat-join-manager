-- +goose Up
ALTER TABLE sessions ADD COLUMN is_estimated_join INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE sessions DROP COLUMN is_estimated_join;
