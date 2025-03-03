-- +goose Up
-- +goose StatementBegin
CREATE TABLE spaces
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    sid        TEXT NOT NULL UNIQUE,
    owner_id   INTEGER NOT NULL,
    name       TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CONSTRAINT fk_users_owner_id FOREIGN KEY (owner_id) REFERENCES users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE spaces;
-- +goose StatementEnd