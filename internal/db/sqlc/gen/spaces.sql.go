// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: spaces.sql

package sqlc

import (
	"context"
)

const findSpaceBySid = `-- name: FindSpaceBySid :one
;

SELECT id, sid, name, visibility, created_at, updated_at, deleted_at
FROM spaces
WHERE deleted_at IS NULL
  AND sid = ? LIMIT 1
`

func (q *Queries) FindSpaceBySid(ctx context.Context, sid string) (Space, error) {
	row := q.db.QueryRowContext(ctx, findSpaceBySid, sid)
	var i Space
	err := row.Scan(
		&i.ID,
		&i.Sid,
		&i.Name,
		&i.Visibility,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const findSpaceBySidAndUserId = `-- name: FindSpaceBySidAndUserId :one
;

SELECT spaces.id, spaces.sid, spaces.name, spaces.visibility, spaces.created_at, spaces.updated_at, spaces.deleted_at, spaces_users.role
FROM spaces
         LEFT JOIN spaces_users ON
    spaces_users.space_id = spaces.id
WHERE deleted_at IS NULL
  AND spaces.sid = ?1
  AND spaces_users.user_id = ?2 LIMIT 1
`

type FindSpaceBySidAndUserIdParams struct {
	SpaceID string `db:"space_id" json:"space_id"`
	UserID  int64  `db:"user_id" json:"user_id"`
}

type FindSpaceBySidAndUserIdRow struct {
	Space Space   `db:"space" json:"space"`
	Role  *string `db:"role" json:"role"`
}

func (q *Queries) FindSpaceBySidAndUserId(ctx context.Context, arg FindSpaceBySidAndUserIdParams) (FindSpaceBySidAndUserIdRow, error) {
	row := q.db.QueryRowContext(ctx, findSpaceBySidAndUserId, arg.SpaceID, arg.UserID)
	var i FindSpaceBySidAndUserIdRow
	err := row.Scan(
		&i.Space.ID,
		&i.Space.Sid,
		&i.Space.Name,
		&i.Space.Visibility,
		&i.Space.CreatedAt,
		&i.Space.UpdatedAt,
		&i.Space.DeletedAt,
		&i.Role,
	)
	return i, err
}

const findSpacesByUserId = `-- name: FindSpacesByUserId :many
;

SELECT DISTINCT spaces.id, spaces.sid, spaces.name, spaces.visibility, spaces.created_at, spaces.updated_at, spaces.deleted_at, spaces_users.role
FROM spaces
         INNER JOIN spaces_users on
    spaces_users.space_id = spaces.id
WHERE spaces.deleted_at IS NULL
  AND spaces_users.user_id = ?1
`

type FindSpacesByUserIdRow struct {
	Space Space  `db:"space" json:"space"`
	Role  string `db:"role" json:"role"`
}

func (q *Queries) FindSpacesByUserId(ctx context.Context, userID int64) ([]FindSpacesByUserIdRow, error) {
	rows, err := q.db.QueryContext(ctx, findSpacesByUserId, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FindSpacesByUserIdRow
	for rows.Next() {
		var i FindSpacesByUserIdRow
		if err := rows.Scan(
			&i.Space.ID,
			&i.Space.Sid,
			&i.Space.Name,
			&i.Space.Visibility,
			&i.Space.CreatedAt,
			&i.Space.UpdatedAt,
			&i.Space.DeletedAt,
			&i.Role,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const insertSpace = `-- name: InsertSpace :one
INSERT INTO spaces (sid,
                    created_at,
                    updated_at,
                    name)
VALUES (?1,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        ?2) RETURNING id, sid, name, visibility, created_at, updated_at, deleted_at
`

type InsertSpaceParams struct {
	Sid  string `db:"sid" json:"sid"`
	Name string `db:"name" json:"name"`
}

func (q *Queries) InsertSpace(ctx context.Context, arg InsertSpaceParams) (Space, error) {
	row := q.db.QueryRowContext(ctx, insertSpace, arg.Sid, arg.Name)
	var i Space
	err := row.Scan(
		&i.ID,
		&i.Sid,
		&i.Name,
		&i.Visibility,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.DeletedAt,
	)
	return i, err
}

const insertSpaceUser = `-- name: InsertSpaceUser :exec
;

INSERT INTO spaces_users (space_id,
                          user_id,
                          role)
VALUES (?1,
        ?2,
        ?3)
`

type InsertSpaceUserParams struct {
	SpaceID int64  `db:"space_id" json:"space_id"`
	UserID  int64  `db:"user_id" json:"user_id"`
	Role    string `db:"role" json:"role"`
}

func (q *Queries) InsertSpaceUser(ctx context.Context, arg InsertSpaceUserParams) error {
	_, err := q.db.ExecContext(ctx, insertSpaceUser, arg.SpaceID, arg.UserID, arg.Role)
	return err
}
