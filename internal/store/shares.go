package store

import (
	"context"
	"time"
)

// Share is the public read-only link of a collection.
type Share struct {
	CollectionID string    `db:"collection_id"`
	Token        string    `db:"token"`
	CreatedBy    string    `db:"created_by"`
	CreatedAt    time.Time `db:"created_at"`
	// DisabledAt: disabled by the instance admins after a report, for DisabledReason
	DisabledAt     *time.Time `db:"disabled_at"`
	DisabledReason *string    `db:"disabled_reason"`
}

const shareColumns = `sh.collection_id, sh.token, sh.created_by, sh.created_at, sh.disabled_at, sh.disabled_reason`

// ShareOf returns ErrNotFound if the collection has no public link; disabled ones are returned.
func (s *Store) ShareOf(ctx context.Context, collectionID string) (*Share, error) {
	var share Share
	return &share, s.get(ctx, &share, `SELECT `+shareColumns+` FROM collection_shares sh WHERE sh.collection_id = ?`,
		collectionID)
}

// ShareByToken returns ErrNotFound for unknown, revoked and disabled links and those of blocked accounts.
func (s *Store) ShareByToken(ctx context.Context, token string) (*Share, error) {
	var share Share
	return &share, s.get(ctx, &share, `SELECT `+shareColumns+` FROM collection_shares sh
		JOIN users u ON u.id = sh.created_by WHERE sh.token = ? AND sh.disabled_at IS NULL AND u.blocked_at IS NULL`, token)
}

// ReplaceShare gives the collection a new link; an existing one stops working.
func (s *Store) ReplaceShare(ctx context.Context, share *Share) error {
	share.CreatedAt = now()
	if err := s.DeleteShare(ctx, share.CollectionID); err != nil {
		return err
	}
	_, err := s.exec(ctx, `INSERT INTO collection_shares (collection_id, token, created_by, created_at) VALUES (?, ?, ?, ?)`,
		share.CollectionID, share.Token, share.CreatedBy, share.CreatedAt)
	return err
}

// DeleteShare keeps disabled links: the space cannot share the collection again by revoking it.
func (s *Store) DeleteShare(ctx context.Context, collectionID string) error {
	_, err := s.exec(ctx, `DELETE FROM collection_shares WHERE collection_id = ? AND disabled_at IS NULL`, collectionID)
	return err
}

// SharedCollections lists the ids of the collections of the space that have a public link.
func (s *Store) SharedCollections(ctx context.Context, spaceID string) (map[string]bool, error) {
	var ids []string
	if err := s.sel(ctx, &ids, `SELECT sh.collection_id FROM collection_shares sh
		JOIN collections c ON c.id = sh.collection_id WHERE c.space_id = ? AND sh.disabled_at IS NULL`, spaceID); err != nil {
		return nil, err
	}
	shared := make(map[string]bool, len(ids))
	for _, id := range ids {
		shared[id] = true
	}
	return shared, nil
}
