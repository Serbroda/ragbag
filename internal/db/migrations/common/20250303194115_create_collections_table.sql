-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections
(
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    sid        TEXT    NOT NULL UNIQUE,
    space_id   INTEGER NOT NULL,
    parent_id  INTEGER,
    name       TEXT    NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CONSTRAINT fk_collections_space_id FOREIGN KEY (space_id) REFERENCES spaces (id),
    CONSTRAINT fk_collections_parent_id FOREIGN KEY (parent_id) REFERENCES collections (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE collections;
-- +goose StatementEnd