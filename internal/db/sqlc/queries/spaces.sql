-- name: InsertSpace :one
INSERT INTO spaces (sid,
                    created_at,
                    updated_at,
                    owner_id,
                    name)
VALUES (sqlc.arg('sid'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('owner_id'),
        sqlc.arg('name')) RETURNING *
;

-- name: FindSpaceBySid :one
SELECT *
FROM spaces u
WHERE sid = ?
  AND deleted_at IS NULL LIMIT 1;

-- name: FindSpacesByOwnerId :many
SELECT *
FROM spaces u
WHERE owner_id = ?
  AND deleted_at IS NULL;
