// Package auth issues and verifies the JWTs and password hashes.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	TypeAccess  = "ACCESS"
	TypeRefresh = "REFRESH"
)

// Claims of both token types; token_version lets "log out everywhere" and password changes revoke them.
type Claims struct {
	Type         string   `json:"type"`
	Username     string   `json:"username,omitempty"`
	TokenVersion int      `json:"token_version"`
	Roles        []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewTokens(secret []byte, issuer string, accessTTL, refreshTTL time.Duration) *Tokens {
	return &Tokens{secret: secret, issuer: issuer, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

func (t *Tokens) RefreshTTL() time.Duration { return t.refreshTTL }

func (t *Tokens) Access(userID, username string, tokenVersion int) (string, error) {
	return t.sign(Claims{Type: TypeAccess, Username: username, TokenVersion: tokenVersion, Roles: []string{"USER"}},
		userID, t.accessTTL)
}

func (t *Tokens) Refresh(userID string, tokenVersion int) (string, error) {
	return t.sign(Claims{Type: TypeRefresh, TokenVersion: tokenVersion}, userID, t.refreshTTL)
}

func (t *Tokens) sign(claims Claims, subject string, ttl time.Duration) (string, error) {
	issuedAt := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    t.issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(ttl)),
		ID:        uuid.NewString(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

// Parse verifies signature, issuer, expiry and the expected token type.
func (t *Tokens) Parse(token, expectedType string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(t.issuer), jwt.WithExpirationRequired())
	if err != nil {
		return nil, err
	}
	if claims.Type != expectedType {
		return nil, fmt.Errorf("expected a %s token", expectedType)
	}
	if claims.Subject == "" {
		return nil, errors.New("token without subject")
	}
	return &claims, nil
}

// --- Passwords ---

const MinPasswordLength = 8

var ErrPasswordTooShort = fmt.Errorf("Passwords must have at least %d characters", MinPasswordLength)

func ValidatePassword(password string) error {
	if len([]rune(password)) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	// BCrypt only uses the first 72 bytes
	if len(password) > 72 {
		return errors.New("Passwords must be at most 72 bytes long")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
