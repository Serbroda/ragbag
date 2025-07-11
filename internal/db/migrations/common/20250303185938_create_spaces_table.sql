-- +goose Up
-- +goose StatementBegin
CREATE TABLE spaces
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    sid        TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL,
    visibility TEXT    NOT NULL DEFAULT 'PRIVATE',
    created_at DATETIME         DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME         DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE spaces_users
(
    space_id INTEGER NOT NULL,
    user_id  INTEGER NOT NULL,
    role     TEXT    NOT NULL,
    PRIMARY KEY (space_id, user_id),
    CONSTRAINT fk_spaces_users_space_id FOREIGN KEY (space_id) REFERENCES spaces (id),
    CONSTRAINT fk_spaces_users_user_id FOREIGN KEY (user_id) REFERENCES users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE spaces_users;
DROP TABLE spaces;
-- +goose StatementEnd