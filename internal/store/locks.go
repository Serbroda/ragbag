package store

import (
	"context"
	"time"
)

// PinState is what unlocking needs: the PIN hash and the brake against guessing.
type PinState struct {
	Hash         *string    `db:"pin_hash"`
	Failures     int        `db:"pin_failures"`
	BlockedUntil *time.Time `db:"pin_blocked_until"`
}

func (s *Store) PinState(ctx context.Context, userID string) (*PinState, error) {
	var state PinState
	return &state, s.get(ctx, &state, `SELECT pin_hash, pin_failures, pin_blocked_until FROM users WHERE id = ?`, userID)
}

// SetPin stores a new PIN hash and forgets earlier failures.
func (s *Store) SetPin(ctx context.Context, userID, hash string) error {
	_, err := s.exec(ctx, `UPDATE users SET pin_hash = ?, pin_failures = 0, pin_blocked_until = NULL, updated_at = ?
		WHERE id = ?`, hash, now(), userID)
	return err
}

// SetPinFailures records wrong PINs; blockedUntil pauses unlocking after too many.
func (s *Store) SetPinFailures(ctx context.Context, userID string, failures int, blockedUntil *time.Time) error {
	if blockedUntil != nil {
		utc := blockedUntil.UTC()
		blockedUntil = &utc
	}
	_, err := s.exec(ctx, `UPDATE users SET pin_failures = ?, pin_blocked_until = ? WHERE id = ?`,
		failures, blockedUntil, userID)
	return err
}

func (s *Store) LockCollection(ctx context.Context, userID, collectionID string) error {
	_, err := s.exec(ctx, `INSERT INTO collection_locks (user_id, collection_id, created_at) VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING`, userID, collectionID, now())
	return err
}

func (s *Store) UnlockCollection(ctx context.Context, userID, collectionID string) error {
	_, err := s.exec(ctx, `DELETE FROM collection_locks WHERE user_id = ? AND collection_id = ?`, userID, collectionID)
	return err
}

// LockedCollectionsOf lists the collections of the space the user locked for themselves.
func (s *Store) LockedCollectionsOf(ctx context.Context, spaceID, userID string) ([]string, error) {
	var ids []string
	err := s.sel(ctx, &ids, `SELECT l.collection_id FROM collection_locks l
		JOIN collections c ON c.id = l.collection_id WHERE c.space_id = ? AND l.user_id = ?`, spaceID, userID)
	return ids, err
}
