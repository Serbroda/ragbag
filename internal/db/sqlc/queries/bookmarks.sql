-- name: InsertBookmark :one
INSERT INTO bookmarks (id,
                       collection_id,
                       url,
                       title,
                       description)
VALUES (sqlc.arg('id'),
        sqlc.arg('collection_id'),
        sqlc.arg('url'),
        sqlc.arg('title'),
        sqlc.arg('description')) RETURNING *
;
