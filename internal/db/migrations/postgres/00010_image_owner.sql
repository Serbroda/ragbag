-- +goose Up
-- Who uploaded an image, for the upload quota (older images count for nobody)
ALTER TABLE images ADD COLUMN created_by TEXT REFERENCES users (id) ON DELETE SET NULL;
CREATE INDEX idx_images_created_by ON images (created_by);

-- +goose Down
DROP INDEX idx_images_created_by;
ALTER TABLE images DROP COLUMN created_by;
