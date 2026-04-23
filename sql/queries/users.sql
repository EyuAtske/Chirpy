-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, hashed_password, email)
VALUES (
    gen_random_uuid(),
    Now(),
    Now(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;