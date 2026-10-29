-- +goose Up
-- When a member last looked at a collection: links others added after that count as new for them
CREATE TABLE collection_visits (
    user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    collection_id TEXT NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    seen_at       TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, collection_id)
);
CREATE INDEX idx_bookmarks_created ON bookmarks (created_at);

-- Emoji reactions from a fixed set, one row per user and emoji
CREATE TABLE bookmark_reactions (
    bookmark_id TEXT NOT NULL REFERENCES bookmarks (id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    emoji       TEXT NOT NULL,
    created_at  TIMESTAMP NOT NULL,
    PRIMARY KEY (bookmark_id, user_id, emoji)
);

-- Personal order and folders of the space bar as JSON (SpaceBarDto), so it follows the user across devices
CREATE TABLE space_bars (
    user_id    TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    layout     TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE space_bars;
DROP TABLE bookmark_reactions;
DROP INDEX idx_bookmarks_created;
DROP TABLE collection_visits;
