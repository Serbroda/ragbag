-- +goose Up
-- Markdown shown to members when they first open the space (no raw HTML)
ALTER TABLE spaces ADD COLUMN welcome_message TEXT;
-- When the member saw the welcome (the owner's introduction in the own first space, else the welcome message);
-- existing memberships count as welcomed
ALTER TABLE space_members ADD COLUMN welcomed_at TIMESTAMP;
UPDATE space_members SET welcomed_at = created_at;

-- +goose Down
ALTER TABLE space_members DROP COLUMN welcomed_at;
ALTER TABLE spaces DROP COLUMN welcome_message;
