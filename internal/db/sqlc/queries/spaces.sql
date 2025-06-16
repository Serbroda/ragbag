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

-- name:

-- name: FindSpaceBySid :one
SELECT *
FROM spaces s
WHERE deleted_at IS NULL
  AND sid = ? LIMIT 1;

-- name: FindSpacesByOwnerId :many
SELECT DISTINCT s.*
FROM spaces s
         LEFT JOIN spaces_users su on
    su.space_id = s.id
WHERE s.deleted_at IS NULL
  AND (
    s.owner_id = sqlc.arg('user_id')
        OR su.user_id = sqlc.arg('user_id')
    );
