-- +goose Up
-- A Discord webhook per space: new links are posted to its channel. The address is a secret (who has it can
-- post there); discord_webhook_failed_at is set when Discord says the webhook is gone, which stops posting.
ALTER TABLE spaces ADD COLUMN discord_webhook_url TEXT;
ALTER TABLE spaces ADD COLUMN discord_webhook_failed_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE spaces DROP COLUMN discord_webhook_failed_at;
ALTER TABLE spaces DROP COLUMN discord_webhook_url;
