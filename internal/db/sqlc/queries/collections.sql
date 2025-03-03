-- name: InsertCollection :one
INSERT INTO collections (sid,
                         created_at,
                         updated_at,
                         space_id,
                         parent_id,
                         name)
VALUES (sqlc.arg('sid'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('space_id'),
        sqlc.arg('parent_id'),
        sqlc.arg('name')) RETURNING *
;
