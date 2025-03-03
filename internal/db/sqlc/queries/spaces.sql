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
