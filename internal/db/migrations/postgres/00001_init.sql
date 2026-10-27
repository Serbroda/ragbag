-- +goose Up
CREATE TABLE users (
    id              TEXT PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    token_version   INTEGER NOT NULL DEFAULT 0,
    avatar_image_id TEXT,
    created_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL
);

-- Avatars and space icons; replaced images are deleted by the application
CREATE TABLE images (
    id           TEXT PRIMARY KEY,
    content_type TEXT NOT NULL,
    data         BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL
);

CREATE TABLE spaces (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT,
    icon_image_id TEXT,
    created_by    TEXT NOT NULL REFERENCES users (id),
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE TABLE space_members (
    space_id   TEXT NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (space_id, user_id)
);
CREATE INDEX idx_space_members_user ON space_members (user_id);

CREATE TABLE collections (
    id          TEXT PRIMARY KEY,
    space_id    TEXT NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    parent_id   TEXT REFERENCES collections (id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    -- Private: only collection_members see it and everything below it
    restricted  BOOLEAN NOT NULL DEFAULT FALSE,
    created_by  TEXT NOT NULL REFERENCES users (id),
    created_at  TIMESTAMPTZ NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_collections_space ON collections (space_id);
CREATE INDEX idx_collections_parent ON collections (parent_id);

CREATE TABLE collection_members (
    collection_id TEXT NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    PRIMARY KEY (collection_id, user_id)
);

CREATE TABLE bookmarks (
    id            TEXT PRIMARY KEY,
    collection_id TEXT NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    url           TEXT NOT NULL,
    title         TEXT,
    description   TEXT,
    og_image      TEXT,
    favicon       TEXT,
    canonical     TEXT,
    created_by    TEXT NOT NULL REFERENCES users (id),
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_bookmarks_collection ON bookmarks (collection_id, created_at);

CREATE TABLE bookmark_tags (
    bookmark_id TEXT NOT NULL REFERENCES bookmarks (id) ON DELETE CASCADE,
    tag         TEXT NOT NULL,
    PRIMARY KEY (bookmark_id, tag)
);

CREATE TABLE bookmark_favorites (
    user_id     TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    bookmark_id TEXT NOT NULL REFERENCES bookmarks (id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, bookmark_id)
);

CREATE TABLE dashboard_sections (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    space_id   TEXT NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    type       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_dashboard_sections_user_space ON dashboard_sections (user_id, space_id);

CREATE TABLE dashboard_items (
    section_id  TEXT NOT NULL REFERENCES dashboard_sections (id) ON DELETE CASCADE,
    bookmark_id TEXT NOT NULL REFERENCES bookmarks (id) ON DELETE CASCADE,
    position    INTEGER NOT NULL,
    PRIMARY KEY (section_id, bookmark_id)
);

CREATE TABLE invites (
    id          TEXT PRIMARY KEY,
    token       TEXT NOT NULL UNIQUE,
    target_type TEXT NOT NULL,
    target_id   TEXT NOT NULL,
    role        TEXT NOT NULL,
    created_by  TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ,
    -- -1: unlimited
    max_uses    INTEGER NOT NULL DEFAULT -1,
    used_count  INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_invites_target ON invites (target_type, target_id);

-- One-time reset links created by instance admins; only a SHA-256 hash of the token is stored
CREATE TABLE password_resets (
    token_hash TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    created_by TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE password_resets;
DROP TABLE invites;
DROP TABLE dashboard_items;
DROP TABLE dashboard_sections;
DROP TABLE bookmark_favorites;
DROP TABLE bookmark_tags;
DROP TABLE bookmarks;
DROP TABLE collection_members;
DROP TABLE collections;
DROP TABLE space_members;
DROP TABLE spaces;
DROP TABLE images;
DROP TABLE users;
