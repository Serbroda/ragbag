-- name: InsertCollection :one
INSERT INTO collections (id,
                         created_at,
                         updated_at,
                         space_id,
                         name)
VALUES (sqlc.arg('id'),
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        sqlc.arg('space_id'),
        sqlc.arg('name')) RETURNING *
;

-- name: GetAllCollections :many
SELECT *
FROM collections
WHERE collections.deleted_at IS NULL
;

-- name: GetCollectionsByUserAndSpace :many
SELECT
    sqlc.embed(collections),
    collections_users.role
FROM collections
         LEFT JOIN collections_users
                   ON collections_users.collection_id = collections.id
                       AND collections_users.user_id = sqlc.arg('user_id')
WHERE collections.deleted_at IS NULL
  AND collections.space_id = sqlc.arg('space_id')
  AND (
    collections.visibility = 'PUBLIC'
        OR collections_users.role IS NOT NULL
    )
;

