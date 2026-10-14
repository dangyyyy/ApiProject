-- +goose Up
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_completed_created_at ON tasks (completed, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_tasks_completed_created_at;
DROP INDEX IF EXISTS idx_tasks_created_at;