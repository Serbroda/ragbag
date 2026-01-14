-- +goose Up
-- +goose StatementBegin
CREATE TABLE spaces
(
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'PRIVATE' CHECK (visibility IN (
                                                                     'PRIVATE', -- only visible for space members
                                                                     'PUBLIC'   -- visible for everyone
        )),
    created_at DATETIME      DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME      DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by TEXT NOT NULL
);

CREATE TABLE spaces_members
(
    space_id TEXT NOT NULL,
    user_id  TEXT NOT NULL,
    role     TEXT NOT NULL CHECK (role IN (
                                           'OWNER',       -- full access including managing members
                                           'CONTRIBUTOR', -- can create and edit content
                                           'VIEWER'       -- read-only access
        )),
    PRIMARY KEY (space_id, user_id),
    CONSTRAINT fk_spaces_members_space_id FOREIGN KEY (space_id) REFERENCES spaces (id),
    CONSTRAINT fk_spaces_members_user_id FOREIGN KEY (user_id) REFERENCES users (id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE spaces_members;
DROP TABLE spaces;
-- +goose StatementEnd
