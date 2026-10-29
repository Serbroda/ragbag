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
}

// ShareOf returns ErrNotFound if the collection has no public link.
func (s *Store) ShareOf(ctx context.Context, collectionID string) (*Share, error) {
	var share Share
	return &share, s.get(ctx, &share, `SELECT collection_id, token, created_by, created_at FROM collection_shares
		WHERE collection_id = ?`, collectionID)
}

// ShareByToken returns ErrNotFound for unknown or revoked links.
func (s *Store) ShareByToken(ctx context.Context, token string) (*Share, error) {
	var share Share
	return &share, s.get(ctx, &share, `SELECT collection_id, token, created_by, created_at FROM collection_shares
		WHERE token = ?`, token)
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

func (s *Store) DeleteShare(ctx context.Context, collectionID string) error {
	_, err := s.exec(ctx, `DELETE FROM collection_shares WHERE collection_id = ?`, collectionID)
	return err
}

// SharedCollections lists the ids of the collections of the space that have a public link.
func (s *Store) SharedCollections(ctx context.Context, spaceID string) (map[string]bool, error) {
	var ids []string
	if err := s.sel(ctx, &ids, `SELECT sh.collection_id FROM collection_shares sh
		JOIN collections c ON c.id = sh.collection_id WHERE c.space_id = ?`, spaceID); err != nil {
		return nil, err
	}
	shared := make(map[string]bool, len(ids))
	for _, id := range ids {
		shared[id] = true
	}
	return shared, nil
}
