package store

import (
	"context"
	"time"
)

// Identity connects an account to the single sign-on provider: who the provider says the person is.
type Identity struct {
	Issuer    string    `db:"issuer"`
	Subject   string    `db:"subject"`
	UserID    string    `db:"user_id"`
	Email     *string   `db:"email"`
	CreatedAt time.Time `db:"created_at"`
}

const identityColumns = `issuer, subject, user_id, email, created_at`

// IdentityBySubject returns ErrNotFound for people the provider knows but no account is connected to.
func (s *Store) IdentityBySubject(ctx context.Context, issuer, subject string) (*Identity, error) {
	var identity Identity
	return &identity, s.get(ctx, &identity, `SELECT `+identityColumns+` FROM user_identities
		WHERE issuer = ? AND subject = ?`, issuer, subject)
}

// IdentityOf returns ErrNotFound while the account is not connected to the provider.
func (s *Store) IdentityOf(ctx context.Context, userID, issuer string) (*Identity, error) {
	var identity Identity
	return &identity, s.get(ctx, &identity, `SELECT `+identityColumns+` FROM user_identities
		WHERE user_id = ? AND issuer = ?`, userID, issuer)
}

func (s *Store) AddIdentity(ctx context.Context, identity Identity) error {
	_, err := s.exec(ctx, `INSERT INTO user_identities (`+identityColumns+`) VALUES (?, ?, ?, ?, ?)`,
		identity.Issuer, identity.Subject, identity.UserID, identity.Email, now())
	return err
}

func (s *Store) RemoveIdentity(ctx context.Context, userID, issuer string) error {
	_, err := s.exec(ctx, `DELETE FROM user_identities WHERE user_id = ? AND issuer = ?`, userID, issuer)
	return err
}
