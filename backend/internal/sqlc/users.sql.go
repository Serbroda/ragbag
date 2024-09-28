// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: users.sql

package sqlc

import (
	"context"
)

const create = `-- name: Create :execlastid
INSERT INTO users (created_at,
                   updated_at,
                   email,
                   username,
                   password)
VALUES (CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP,
        ?,
        ?,
        ?)
`

type CreateParams struct {
	Email    string `db:"email" json:"email"`
	Username string `db:"username" json:"username"`
	Password string `db:"password" json:"password"`
}

func (q *Queries) Create(ctx context.Context, arg CreateParams) (int64, error) {
	result, err := q.db.ExecContext(ctx, create, arg.Email, arg.Username, arg.Password)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

const findByEmail = `-- name: FindByEmail :one
SELECT id, email, username, password, created_at, updated_at
FROM users u
WHERE lower(username) = ? LIMIT 1
`

func (q *Queries) FindByEmail(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRowContext(ctx, findByEmail, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const findById = `-- name: FindById :one
SELECT id, email, username, password, created_at, updated_at
FROM users u
WHERE id = ? LIMIT 1
`

func (q *Queries) FindById(ctx context.Context, id int64) (User, error) {
	row := q.db.QueryRowContext(ctx, findById, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const findByUsername = `-- name: FindByUsername :one
SELECT id, email, username, password, created_at, updated_at
FROM users u
WHERE lower(username) = ? LIMIT 1
`

func (q *Queries) FindByUsername(ctx context.Context, username string) (User, error) {
	row := q.db.QueryRowContext(ctx, findByUsername, username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Username,
		&i.Password,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
