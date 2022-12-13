-- name: FindPage :one
SELECT *
FROM pages
WHERE id = ?
LIMIT 1;
-- name: FindPageByShortId :one
SELECT *
FROM pages
WHERE short_id = ?
LIMIT 1;
-- name: CreatePage :execlastid
INSERT INTO pages (
        created_at,
        short_id,
        owner_id,
        space_id,
        parent_id,
        name,
        description,
        visibility
    )
VALUES(
        CURRENT_TIMESTAMP,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?,
        ?
    );
-- name: UpdatePage :exec
UPDATE pages
SET updated_at = CURRENT_TIMESTAMP,
    owner_id = COALESCE(?, owner_id),
    parent_id = COALESCE(?, parent_id),
    name = COALESCE(?, name),
    description = COALESCE(?, description),
    visibility = COALESCE(?, visibility)
WHERE id = ?;
-- name: InsertUserPage :exec
INSERT INTO users_pages (user_id, page_id, role_id, created_at)
VALUES(?, ?, ?, CURRENT_TIMESTAMP);
-- name: DeleteUserPage :exec
DELETE FROM users_pages
WHERE user_id = ?
    AND page_id = ?;