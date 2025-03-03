-- name: InsertBookmark :one
INSERT INTO bookmarks (sid,
                       created_at,
                       updated_at,
                       collection_id,
                       url,
                       title,
                       description)
VALUES (sqlc.arg('sid'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('collection_id'),
        sqlc.arg('url'),
        sqlc.arg('title'),
        sqlc.arg('description')) RETURNING *
;
