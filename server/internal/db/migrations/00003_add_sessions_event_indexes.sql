-- +goose Up
CREATE INDEX IF NOT EXISTS idx_sessions_join_event  ON sessions(join_event_id);
CREATE INDEX IF NOT EXISTS idx_sessions_leave_event ON sessions(leave_event_id);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_leave_event;
DROP INDEX IF EXISTS idx_sessions_join_event;
