-- name: InsertUser :one
INSERT INTO users (sid,
                   created_at,
                   updated_at,
                   email,
                   password)
VALUES (sqlc.arg('sid'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('email'),
        sqlc.arg('password')) RETURNING *
;

-- name: FindUserBySid :one
SELECT *
FROM users
WHERE sid = ?
  AND deleted_at IS NULL LIMIT 1;

-- name: FindUserByEmail :one
SELECT *
FROM users u
WHERE email = ?
  AND deleted_at IS NULL LIMIT 1;

-- name: UpdateUser :exec
UPDATE users
SET password = COALESCE(sqlc.arg('password'), password)
WHERE sid = ?
  AND deleted_at IS NULL
;

-- name: DeleteUserSoft :exec
UPDATE users
SET deleted_at = CURRENT_TIMESTAMP
WHERE sid = ?
;
