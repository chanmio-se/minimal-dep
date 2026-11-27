-- name: CreateTask :exec
INSERT INTO tasks (id, owner_id, title, done, created_at)
VALUES (?, ?, ?, ?, ?);

-- name: ListTasksByOwner :many
SELECT id, owner_id, title, done, created_at
FROM tasks
WHERE owner_id = ?
ORDER BY created_at, id;

-- name: FindTaskByID :one
SELECT id, owner_id, title, done, created_at
FROM tasks
WHERE id = ?;

-- name: SaveTask :exec
UPDATE tasks
SET title = ?, done = ?, created_at = ?
WHERE id = ?;
