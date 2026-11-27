-- name: CreateUser :exec
INSERT INTO identity_users (id, email, name, created_at)
VALUES (?, ?, ?, ?);

-- name: FindUserByEmail :one
SELECT id, email, name, created_at
FROM identity_users
WHERE email = ?;

-- name: FindUserByID :one
SELECT id, email, name, created_at
FROM identity_users
WHERE id = ?;
