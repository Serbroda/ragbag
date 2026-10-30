package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type User struct {
	ID            string    `db:"id"`
	Username      string    `db:"username"`
	Email         string    `db:"email"`
	PasswordHash  string    `db:"password_hash"`
	TokenVersion  int       `db:"token_version"`
	AvatarImageID *string   `db:"avatar_image_id"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

const userColumns = `id, username, email, password_hash, token_version, avatar_image_id, created_at, updated_at`

func (s *Store) CreateUser(ctx context.Context, username, email, passwordHash string) (*User, error) {
	at := now()
	user := &User{ID: newID(), Username: username, Email: email, PasswordHash: passwordHash, CreatedAt: at, UpdatedAt: at}
	_, err := s.exec(ctx, `INSERT INTO users (`+userColumns+`) VALUES (?, ?, ?, ?, 0, NULL, ?, ?)`,
		user.ID, user.Username, user.Email, user.PasswordHash, at, at)
	return user, err
}

func (s *Store) UserByID(ctx context.Context, id string) (*User, error) {
	var user User
	return &user, s.get(ctx, &user, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
}

// UserByLogin finds a user by username or email, both stored lower-case.
func (s *Store) UserByLogin(ctx context.Context, login string) (*User, error) {
	var user User
	login = strings.ToLower(strings.TrimSpace(login))
	return &user, s.get(ctx, &user, `SELECT `+userColumns+` FROM users WHERE username = ? OR email = ?`, login, login)
}

func (s *Store) UsernameOrEmailTaken(ctx context.Context, username, email string) (bool, error) {
	var count int
	err := s.get(ctx, &count, `SELECT COUNT(*) FROM users WHERE username IN (?, ?) OR email IN (?, ?)`,
		username, email, username, email)
	return count > 0, err
}

func (s *Store) SetPassword(ctx context.Context, userID, passwordHash string) error {
	_, err := s.exec(ctx, `UPDATE users SET password_hash = ?, token_version = token_version + 1, updated_at = ? WHERE id = ?`,
		passwordHash, now(), userID)
	return err
}

// BumpTokenVersion invalidates every access and refresh token issued so far.
func (s *Store) BumpTokenVersion(ctx context.Context, userID string) error {
	_, err := s.exec(ctx, `UPDATE users SET token_version = token_version + 1, updated_at = ? WHERE id = ?`, now(), userID)
	return err
}

func (s *Store) SetAvatar(ctx context.Context, userID string, imageID *string) error {
	_, err := s.exec(ctx, `UPDATE users SET avatar_image_id = ?, updated_at = ? WHERE id = ?`, imageID, now(), userID)
	return err
}

type UserWithSpaces struct {
	User
	Spaces int `db:"spaces"`
}

func (s *Store) AllUsers(ctx context.Context) ([]UserWithSpaces, error) {
	var users []UserWithSpaces
	err := s.sel(ctx, &users, `
		SELECT u.id, u.username, u.email, u.password_hash, u.token_version, u.avatar_image_id, u.created_at, u.updated_at,
		       (SELECT COUNT(*) FROM space_members m WHERE m.user_id = u.id) AS spaces
		FROM users u ORDER BY u.username`)
	return users, err
}

func (s *Store) UsersByIDs(ctx context.Context, ids []string) ([]User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var users []User
	err := s.selIn(ctx, &users, `SELECT `+userColumns+` FROM users WHERE id IN (?) ORDER BY username`, ids)
	return users, err
}

// --- Images ---

type Image struct {
	ID          string    `db:"id"`
	ContentType string    `db:"content_type"`
	Data        []byte    `db:"data"`
	CreatedAt   time.Time `db:"created_at"`
}

// StartSpace returns the space the user opens on start, or nil for the last visited one.
func (s *Store) StartSpace(ctx context.Context, userID string) (*string, error) {
	var ids []sql.NullString
	if err := s.sel(ctx, &ids, `SELECT start_space_id FROM users WHERE id = ?`, userID); err != nil || len(ids) == 0 ||
		!ids[0].Valid {
		return nil, err
	}
	return &ids[0].String, nil
}

func (s *Store) SetStartSpace(ctx context.Context, userID string, spaceID *string) error {
	_, err := s.exec(ctx, `UPDATE users SET start_space_id = ?, updated_at = ? WHERE id = ?`, spaceID, now(), userID)
	return err
}

func (s *Store) CreateImage(ctx context.Context, contentType string, data []byte) (string, error) {
	id := newID()
	_, err := s.exec(ctx, `INSERT INTO images (id, content_type, data, created_at) VALUES (?, ?, ?, ?)`,
		id, contentType, data, now())
	return id, err
}

func (s *Store) ImageByID(ctx context.Context, id string) (*Image, error) {
	var image Image
	return &image, s.get(ctx, &image, `SELECT id, content_type, data, created_at FROM images WHERE id = ?`, id)
}

// DeleteImage accepts nil so callers can pass the previous image id as is.
func (s *Store) DeleteImage(ctx context.Context, id *string) error {
	if id == nil {
		return nil
	}
	_, err := s.exec(ctx, `DELETE FROM images WHERE id = ?`, *id)
	return err
}

// --- Password resets ---

type PasswordReset struct {
	TokenHash string    `db:"token_hash"`
	UserID    string    `db:"user_id"`
	CreatedBy *string   `db:"created_by"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

// ReplacePasswordReset drops older links of the user (and expired ones of everybody) and stores the new one.
func (s *Store) ReplacePasswordReset(ctx context.Context, reset PasswordReset) error {
	if _, err := s.exec(ctx, `DELETE FROM password_resets WHERE user_id = ? OR expires_at < ?`, reset.UserID, now()); err != nil {
		return err
	}
	_, err := s.exec(ctx, `INSERT INTO password_resets (token_hash, user_id, created_by, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		reset.TokenHash, reset.UserID, reset.CreatedBy, reset.ExpiresAt, now())
	return err
}

func (s *Store) PasswordResetByHash(ctx context.Context, tokenHash string) (*PasswordReset, error) {
	var reset PasswordReset
	return &reset, s.get(ctx, &reset,
		`SELECT token_hash, user_id, created_by, expires_at, created_at FROM password_resets WHERE token_hash = ?`, tokenHash)
}

func (s *Store) DeletePasswordReset(ctx context.Context, tokenHash string) error {
	_, err := s.exec(ctx, `DELETE FROM password_resets WHERE token_hash = ?`, tokenHash)
	return err
}
