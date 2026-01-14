-- name: InsertSpace :one
INSERT INTO spaces (
                    id,
                    name,
                    created_by
)
VALUES (
        sqlc.arg('id'),
        sqlc.arg('name'),
        sqlc.arg('created_by')
       ) RETURNING *
;

-- name: InsertSpaceMember :exec
INSERT INTO spaces_members (space_id,
                            user_id,
                            role)
VALUES (sqlc.arg('space_id'),
        sqlc.arg('user_id'),
        sqlc.arg('role'))
;

-- name: FindSpacesByUserId :many
SELECT DISTINCT sqlc.embed(spaces),
                spaces_members.role
FROM spaces
         INNER JOIN spaces_members ON
    spaces_members.space_id = spaces.id
WHERE spaces.deleted_at IS NULL
  AND spaces_members.user_id = sqlc.arg('user_id')
;

-- name: FindSpaceByIdAndUserId :one
SELECT DISTINCT sqlc.embed(spaces),
                CASE
                    WHEN spaces_members.user_id IS NOT NULL THEN spaces_members.role
                    ELSE 'VIEWER'
                END AS user_role
FROM spaces
         LEFT JOIN spaces_members
                   ON spaces_members.space_id = spaces.id
                       AND spaces_members.user_id = sqlc.arg('user_id')
WHERE spaces.id = sqlc.arg('space_id')
  AND spaces.deleted_at IS NULL
  AND (
    spaces.visibility = 'PUBLIC'
        OR spaces_members.user_id IS NOT NULL
    )
    LIMIT 1
;
