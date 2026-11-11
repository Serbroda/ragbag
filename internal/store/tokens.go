package store

import (
	"context"
	"time"
)

// APIToken is a personal token for scripts and apps; only its hash is stored.
type APIToken struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	Name      string `db:"name"`
	TokenHash string `db:"token_hash"`
	// Scope is READ or WRITE
	Scope      string     `db:"scope"`
	CreatedAt  time.Time  `db:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
	ExpiresAt  *time.Time `db:"expires_at"`
}

const apiTokenColumns = `id, user_id, name, token_hash, scope, created_at, last_used_at, expires_at`

func (s *Store) CreateAPIToken(ctx context.Context, token *APIToken) error {
	token.ID, token.CreatedAt = newID(), now()
	if token.ExpiresAt != nil {
		utc := token.ExpiresAt.UTC()
		token.ExpiresAt = &utc
	}
	_, err := s.exec(ctx, `INSERT INTO api_tokens (`+apiTokenColumns+`) VALUES (?, ?, ?, ?, ?, ?, NULL, ?)`,
		token.ID, token.UserID, token.Name, token.TokenHash, token.Scope, token.CreatedAt, token.ExpiresAt)
	return err
}

// APITokensOf lists the user's tokens, newest first.
func (s *Store) APITokensOf(ctx context.Context, userID string) ([]APIToken, error) {
	var tokens []APIToken
	err := s.sel(ctx, &tokens, `SELECT `+apiTokenColumns+` FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC, id`, userID)
	return tokens, err
}

func (s *Store) APITokenByHash(ctx context.Context, hash string) (*APIToken, error) {
	var token APIToken
	return &token, s.get(ctx, &token, `SELECT `+apiTokenColumns+` FROM api_tokens WHERE token_hash = ?`, hash)
}

// DeleteAPIToken revokes a token of the user and reports whether there was one.
func (s *Store) DeleteAPIToken(ctx context.Context, userID, id string) (bool, error) {
	rows, err := s.exec(ctx, `DELETE FROM api_tokens WHERE id = ? AND user_id = ?`, id, userID)
	return rows > 0, err
}

// TouchAPIToken remembers that the token was just used.
func (s *Store) TouchAPIToken(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, now(), id)
	return err
}
