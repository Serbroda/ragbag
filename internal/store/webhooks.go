package store

import (
	"context"
	"time"
)

// SpaceWebhook is the Discord webhook of a space; URL is nil without one.
type SpaceWebhook struct {
	URL *string `db:"discord_webhook_url"`
	// FailedAt: Discord answered that the webhook no longer exists; nothing is posted until a new one is set
	FailedAt *time.Time `db:"discord_webhook_failed_at"`
}

func (s *Store) SpaceWebhook(ctx context.Context, spaceID string) (*SpaceWebhook, error) {
	var webhook SpaceWebhook
	return &webhook, s.get(ctx, &webhook,
		`SELECT discord_webhook_url, discord_webhook_failed_at FROM spaces WHERE id = ?`, spaceID)
}

// SetSpaceWebhook stores a new address (nil removes it) and forgets an earlier failure.
func (s *Store) SetSpaceWebhook(ctx context.Context, spaceID string, url *string) error {
	_, err := s.exec(ctx, `UPDATE spaces SET discord_webhook_url = ?, discord_webhook_failed_at = NULL, updated_at = ?
		WHERE id = ?`, url, now(), spaceID)
	return err
}

// FailSpaceWebhook stops posting to this address; a newer one set meanwhile is left alone.
func (s *Store) FailSpaceWebhook(ctx context.Context, spaceID, url string) error {
	_, err := s.exec(ctx, `UPDATE spaces SET discord_webhook_failed_at = ? WHERE id = ? AND discord_webhook_url = ?`,
		now(), spaceID, url)
	return err
}
