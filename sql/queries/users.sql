-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: CheckPassword :one
SELECT hashed_password
FROM users
WHERE email = $1;


-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password, is_chirpy_red
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, created_at, updated_at, email, hashed_password, is_chirpy_red
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $2,
    hashed_password = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING id, created_at, updated_at, email, hashed_password, is_chirpy_red;

-- name: GetUserFromRefreshToken :one
SELECT users.* FROM users
JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1
AND revoked_at IS NULL
AND expires_at > NOW();

-- name: UpgradeUserToChirpyRed :one
UPDATE users
SET is_chirpy_red = true,
    updated_at = NOW()
WHERE id = $1
RETURNING id;

