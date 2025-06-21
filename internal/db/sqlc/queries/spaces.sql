-- name: InsertSpace :one
INSERT INTO spaces (sid,
                    created_at,
                    updated_at,
                    name)
VALUES (sqlc.arg('sid'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('name')) RETURNING *
;

-- name: InsertSpaceUser :exec
INSERT INTO spaces_users (space_id,
                          user_id,
                          role)
VALUES (sqlc.arg('space_id'),
        sqlc.arg('user_id'),
        sqlc.arg('role'))
;

-- name: FindSpaceBySid :one
SELECT *
FROM spaces
WHERE deleted_at IS NULL
  AND sid = ? LIMIT 1
;

-- name: FindSpaceBySidAndUserId :one
SELECT sqlc.embed(spaces), spaces_users.role
FROM spaces
         LEFT JOIN spaces_users ON
    spaces_users.space_id = spaces.id
WHERE deleted_at IS NULL
  AND spaces.sid = sqlc.arg('space_id')
  AND spaces_users.user_id = sqlc.arg('user_id') LIMIT 1
;

-- name: FindSpacesByUserId :many
SELECT DISTINCT sqlc.embed(spaces), spaces_users.role
FROM spaces
         INNER JOIN spaces_users on
    spaces_users.space_id = spaces.id
WHERE spaces.deleted_at IS NULL
  AND spaces_users.user_id = sqlc.arg('user_id')
;
