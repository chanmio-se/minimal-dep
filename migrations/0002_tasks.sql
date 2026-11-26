CREATE TABLE IF NOT EXISTS tasks (
    id         TEXT PRIMARY KEY,
    owner_id   TEXT NOT NULL,
    title      TEXT NOT NULL,
    done       INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_owner ON tasks (owner_id, created_at);
