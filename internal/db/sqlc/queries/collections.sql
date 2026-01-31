-- name: InsertCollection :one
INSERT INTO collections (id,
                         space_id,
                         name,
                         created_by)
VALUES (sqlc.arg('id'),
        sqlc.arg('space_id'),
        sqlc.arg('name'),
        sqlc.arg('created_by')) RETURNING *
;

-- name: InsertCollectionFollower :exec
INSERT INTO collections_followers (collection_id,
                                   user_id)
VALUES (sqlc.arg('collection_id'),
        sqlc.arg('user_id'))
;

-- name: FindCollectionsBySpaceIdAndUserId :many
SELECT DISTINCT collections.*
FROM collections
         INNER JOIN spaces_members
                    ON collections.space_id = spaces_members.space_id
                        AND spaces_members.user_id = sqlc.arg('user_id')
WHERE collections.deleted_at IS NULL
  AND collections.space_id = sqlc.arg('space_id')
  AND collections.visibility IN ('INTERNAL', 'PUBLIC')
;

-- name: FindCollectionByIdAndUserId :one
SELECT DISTINCT sqlc.embed(collections),
                CASE
                    WHEN spaces_members.role IS NOT NULL THEN spaces_members.role
                    ELSE 'VIEWER'
                    END AS role
FROM collections
         LEFT JOIN spaces_members
                   ON collections.space_id = spaces_members.space_id
                       AND spaces_members.user_id = sqlc.arg('user_id')
         LEFT JOIN collections_followers
                   ON collections_followers.collection_id = collections.id
                       AND collections_followers.user_id = sqlc.arg('user_id')
WHERE collections.deleted_at IS NULL
  AND collections.id = sqlc.arg('id')
  AND (
    (spaces_members.user_id IS NOT NULL AND collections.visibility IN ('INTERNAL', 'PUBLIC'))
        OR
    (collections_followers.user_id IS NOT NULL AND collections.visibility = 'PUBLIC')
    )
    LIMIT 1
;

-- name: FindFollowingCollectionsByUserId :many
SELECT DISTINCT collections.*
FROM collections
         INNER JOIN collections_followers ON
    collections_followers.collection_id = collections.id AND
    collections_followers.user_id = sqlc.arg('user_id')
WHERE collections.deleted_at IS NULL
  AND collections.visibility = 'PUBLIC'
;
