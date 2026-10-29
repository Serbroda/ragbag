-- +goose Up
-- Public read-only links: anybody with the token sees the collection and its sub-collections, except private ones
CREATE TABLE collection_shares (
    collection_id TEXT PRIMARY KEY REFERENCES collections (id) ON DELETE CASCADE,
    token         TEXT NOT NULL UNIQUE,
    created_by    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE collection_shares;
