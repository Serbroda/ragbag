-- name: InsertBookmark :one
INSERT INTO bookmarks (id,
                       created_at,
                       updated_at,
                       collection_id,
                       url,
                       title,
                       description)
VALUES (sqlc.arg('id'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('collection_id'),
        sqlc.arg('url'),
        sqlc.arg('title'),
        sqlc.arg('description')) RETURNING *
;
