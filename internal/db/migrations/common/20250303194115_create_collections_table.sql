-- +goose Up
-- +goose StatementBegin
CREATE TABLE collections
(
    id         TEXT PRIMARY KEY,
    space_id   TEXT NOT NULL,
    parent_id  TEXT,
    name       TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'INTERNAL' CHECK (visibility IN (
                                                                      'PRIVATE',  -- only visible for owner
                                                                      'INTERNAL', -- visible for all space members
                                                                      'PUBLIC'    -- visible for everyone and followable
        )),
    created_at DATETIME      DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME      DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT NOT NULL,
    CONSTRAINT fk_collections_space_id FOREIGN KEY (space_id) REFERENCES spaces (id),
    CONSTRAINT fk_collections_parent_id FOREIGN KEY (parent_id) REFERENCES collections (id)
);

CREATE TABLE collections_followers
(
    collection_id TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (collection_id, user_id),
    CONSTRAINT fk_collections_followers_collection_id FOREIGN KEY (collection_id) REFERENCES collections (id),
    CONSTRAINT fk_collections_followers_user_id FOREIGN KEY (user_id) REFERENCES users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE collections_followers;
DROP TABLE collections;
-- +goose StatementEnd
