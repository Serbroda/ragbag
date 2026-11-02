package store

import (
	"context"
	"time"
)

type EmailVerification struct {
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// ReplaceEmailVerification stores a new confirmation link for the user; an older one stops working, the invite
// the account registered with stays.
func (s *Store) ReplaceEmailVerification(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	changed, err := s.exec(ctx, `UPDATE email_verifications SET token_hash = ?, expires_at = ?, created_at = ? WHERE user_id = ?`,
		tokenHash, expiresAt.UTC(), now(), userID)
	if err != nil || changed > 0 {
		return err
	}
	_, err = s.exec(ctx, `INSERT INTO email_verifications (user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?)`,
		userID, tokenHash, expiresAt.UTC(), now())
	return err
}

// SetVerificationInvite remembers the space invite an unconfirmed account registered with.
func (s *Store) SetVerificationInvite(ctx context.Context, userID, inviteID string) error {
	_, err := s.exec(ctx, `UPDATE email_verifications SET invite_id = ? WHERE user_id = ?`, inviteID, userID)
	return err
}

// VerificationInvite is the invite an unconfirmed account registered with, empty without one.
func (s *Store) VerificationInvite(ctx context.Context, userID string) (string, error) {
	var inviteID []string
	err := s.sel(ctx, &inviteID, `SELECT invite_id FROM email_verifications WHERE user_id = ? AND invite_id IS NOT NULL`, userID)
	if err != nil || len(inviteID) == 0 {
		return "", err
	}
	return inviteID[0], nil
}

func (s *Store) EmailVerificationByHash(ctx context.Context, tokenHash string) (*EmailVerification, error) {
	var verification EmailVerification
	return &verification, s.get(ctx, &verification,
		`SELECT user_id, token_hash, expires_at, created_at FROM email_verifications WHERE token_hash = ?`, tokenHash)
}

// SetEmailVerified confirms the account and uses up its confirmation link.
func (s *Store) SetEmailVerified(ctx context.Context, userID string) error {
	if _, err := s.exec(ctx, `DELETE FROM email_verifications WHERE user_id = ?`, userID); err != nil {
		return err
	}
	at := now()
	_, err := s.exec(ctx, `UPDATE users SET email_verified_at = ?, updated_at = ? WHERE id = ?`, at, at, userID)
	return err
}

// DeleteUnverifiedUsers removes accounts that were never confirmed. They own nothing yet: the first space is
// created with the confirmation.
func (s *Store) DeleteUnverifiedUsers(ctx context.Context, createdBefore time.Time) (int64, error) {
	return s.exec(ctx, `DELETE FROM users WHERE email_verified_at IS NULL AND created_at < ?`, createdBefore.UTC())
}
