package store

import (
	"context"
	"errors"
	"time"
)

// TwoFactor is a user's second factor: the secret of their authenticator app and the brake against guessing.
type TwoFactor struct {
	// Secret is encrypted by the server; set but not enabled while the user still has to confirm the first code
	Secret    *string    `db:"totp_secret"`
	EnabledAt *time.Time `db:"totp_enabled_at"`
	// Recovery holds the hashes of the unused recovery codes as a JSON array
	Recovery *string `db:"totp_recovery"`
	// LastStep is the time window of the last accepted code, so each one works only once
	LastStep     int64      `db:"totp_last_step"`
	Failures     int        `db:"totp_failures"`
	BlockedUntil *time.Time `db:"totp_blocked_until"`
}

func (s *Store) TwoFactorState(ctx context.Context, userID string) (*TwoFactor, error) {
	var state TwoFactor
	return &state, s.get(ctx, &state, `SELECT totp_secret, totp_enabled_at, totp_recovery, totp_last_step, totp_failures,
		totp_blocked_until FROM users WHERE id = ?`, userID)
}

// StartTwoFactor stores a new secret that waits for its first code; nothing is asked for at login yet.
func (s *Store) StartTwoFactor(ctx context.Context, userID, secret string) error {
	_, err := s.exec(ctx, `UPDATE users SET totp_secret = ?, totp_enabled_at = NULL, totp_recovery = NULL,
		totp_last_step = 0, totp_failures = 0, totp_blocked_until = NULL WHERE id = ?`, secret, userID)
	return err
}

// EnableTwoFactor turns the second factor on, with the hashes of its recovery codes.
func (s *Store) EnableTwoFactor(ctx context.Context, userID, recovery string) error {
	at := now()
	_, err := s.exec(ctx, `UPDATE users SET totp_enabled_at = ?, totp_recovery = ?, updated_at = ? WHERE id = ?`,
		at, recovery, at, userID)
	return err
}

// ClearTwoFactor turns it off and forgets the secret and the recovery codes.
func (s *Store) ClearTwoFactor(ctx context.Context, userID string) error {
	_, err := s.exec(ctx, `UPDATE users SET totp_secret = NULL, totp_enabled_at = NULL, totp_recovery = NULL,
		totp_last_step = 0, totp_failures = 0, totp_blocked_until = NULL, updated_at = ? WHERE id = ?`, now(), userID)
	return err
}

// TwoFactorPassed records an accepted code: its window, the recovery codes left, and no failures anymore.
func (s *Store) TwoFactorPassed(ctx context.Context, userID string, step int64, recovery *string) error {
	_, err := s.exec(ctx, `UPDATE users SET totp_last_step = ?, totp_recovery = ?, totp_failures = 0,
		totp_blocked_until = NULL WHERE id = ?`, step, recovery, userID)
	return err
}

// SetTwoFactorFailures records wrong codes; blockedUntil pauses checking after too many.
func (s *Store) SetTwoFactorFailures(ctx context.Context, userID string, failures int, blockedUntil *time.Time) error {
	if blockedUntil != nil {
		utc := blockedUntil.UTC()
		blockedUntil = &utc
	}
	_, err := s.exec(ctx, `UPDATE users SET totp_failures = ?, totp_blocked_until = ? WHERE id = ?`,
		failures, blockedUntil, userID)
	return err
}

// ClearTwoFactorOf is for the command line (ragbag reset-2fa): it reports whether the account exists.
func (s *Store) ClearTwoFactorOf(ctx context.Context, username string) (bool, error) {
	user, err := s.UserByLogin(ctx, username)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, s.ClearTwoFactor(ctx, user.ID)
}
