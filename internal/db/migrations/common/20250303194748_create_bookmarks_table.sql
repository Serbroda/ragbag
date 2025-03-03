-- +goose Up
-- +goose StatementBegin
CREATE TABLE bookmarks
(
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    sid           TEXT    NOT NULL UNIQUE,
    collection_id INTEGER NOT NULL,
    url           TEXT    NOT NULL,
    title         TEXT,
    description   TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at    DATETIME,
    CONSTRAINT fk_bookmarks_collection_id FOREIGN KEY (collection_id) REFERENCES collections (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE bookmarks;
-- +goose StatementEnd