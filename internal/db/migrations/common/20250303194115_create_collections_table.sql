-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections
(
    id         TEXT PRIMARY KEY,
    space_id   TEXT NOT NULL,
    name       TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'PUBLIC',
    created_at DATETIME      DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME      DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    CONSTRAINT fk_collections_space_id FOREIGN KEY (space_id) REFERENCES spaces (id)
);

CREATE TABLE collections_users
(
    collection_id TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    role          TEXT NOT NULL,
    PRIMARY KEY (collection_id, user_id),
    CONSTRAINT fk_collections_users_collection_id FOREIGN KEY (collection_id) REFERENCES collections (id),
    CONSTRAINT fk_collections_users_user_id FOREIGN KEY (user_id) REFERENCES users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE collections_users;
DROP TABLE collections;
-- +goose StatementEnd