package store

import (
	"context"
	"errors"
	"time"
)

// TermsChange: from EffectiveAt, the terms of use have to be accepted again by accounts that accepted them
// before (or never).
type TermsChange struct {
	ID          string    `db:"id"`
	EffectiveAt time.Time `db:"effective_at"`
	CreatedBy   *string   `db:"created_by"`
	CreatedAt   time.Time `db:"created_at"`
}

// TermsInForceSince is the newest change that is in effect at the given time, nil without one.
func (s *Store) TermsInForceSince(ctx context.Context, at time.Time) (*time.Time, error) {
	var since time.Time
	err := s.get(ctx, &since, `SELECT effective_at FROM terms_changes WHERE effective_at <= ?
		ORDER BY effective_at DESC LIMIT 1`, at.UTC())
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return &since, err
}

// NextTermsChange is the earliest change after the given time, nil without one.
func (s *Store) NextTermsChange(ctx context.Context, after time.Time) (*time.Time, error) {
	var next time.Time
	err := s.get(ctx, &next, `SELECT effective_at FROM terms_changes WHERE effective_at > ?
		ORDER BY effective_at LIMIT 1`, after.UTC())
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	return &next, err
}

func (s *Store) AddTermsChange(ctx context.Context, effectiveAt time.Time, createdBy string) error {
	_, err := s.exec(ctx, `INSERT INTO terms_changes (id, effective_at, created_by, created_at) VALUES (?, ?, ?, ?)`,
		newID(), effectiveAt.UTC().Truncate(time.Microsecond), createdBy, now())
	return err
}

// DeleteTermsChangesAfter withdraws the changes that are not in effect yet and reports whether there were any.
func (s *Store) DeleteTermsChangesAfter(ctx context.Context, after time.Time) (bool, error) {
	rows, err := s.exec(ctx, `DELETE FROM terms_changes WHERE effective_at > ?`, after.UTC())
	return rows > 0, err
}

// CountTermsAccepted counts the active accounts that accepted the terms since the given time, and all of them.
func (s *Store) CountTermsAccepted(ctx context.Context, since time.Time) (accepted, total int, err error) {
	var counts struct {
		Accepted int `db:"accepted"`
		Total    int `db:"total"`
	}
	err = s.get(ctx, &counts, `SELECT
		COALESCE(SUM(CASE WHEN terms_accepted_at >= ? THEN 1 ELSE 0 END), 0) AS accepted, COUNT(*) AS total
		FROM users WHERE deleted_at IS NULL AND blocked_at IS NULL AND email_verified_at IS NOT NULL`, since.UTC())
	return counts.Accepted, counts.Total, err
}
