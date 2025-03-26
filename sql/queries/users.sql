-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, name)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE name = $1;

-- name: ResetUsers :exec
TRUNCATE users CASCADE;

-- name: GetUsers :many
SELECT name FROM users;

-- name: GetUserFromID :one
SELECT name
FROM users
WHERE id = $1;