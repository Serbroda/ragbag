package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

const passwordResetValidity = 24 * time.Hour

// FeatureFlags also reports the version, which the frontend shows in the account menu.
func (s *Server) FeatureFlags(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"registration_enabled": strconv.FormatBool(s.cfg.RegistrationOpen),
		"version":              s.version,
	})
}

func (s *Server) Register(c echo.Context) error {
	if !s.cfg.RegistrationOpen {
		return forbidden("Registration is disabled")
	}
	var req api.RegisterRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if err := auth.ValidatePassword(req.Password); err != nil {
		return badRequest("%s", err.Error())
	}
	user, err := s.createUser(c, req.Username, req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.userDto(user))
}

// createUser registers an account with its own first space, like Discord gives you a server.
func (s *Server) createUser(c echo.Context, username, email, password string) (*store.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	if username == "" || len(username) > 80 || strings.ContainsFunc(username, unicode.IsSpace) {
		return nil, badRequest("Usernames must have 1 to 80 characters without spaces")
	}
	if !strings.Contains(email, "@") || len(email) > 250 {
		return nil, badRequest("Invalid email address")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	var user *store.User
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		taken, err := tx.UsernameOrEmailTaken(ctxOf(c), username, email)
		if err != nil {
			return err
		}
		if taken {
			return conflict("User with same username or email already exists")
		}
		if user, err = tx.CreateUser(ctxOf(c), username, email, hash); err != nil {
			return err
		}
		space, err := tx.CreateSpace(ctxOf(c), capitalize(username)+"'s Space", nil, user.ID)
		if err != nil {
			return err
		}
		return tx.AddMember(ctxOf(c), space.ID, user.ID, string(RoleOwner))
	})
	return user, err
}

func capitalize(value string) string {
	runes := []rune(value)
	if len(runes) > 0 {
		runes[0] = unicode.ToUpper(runes[0])
	}
	return string(runes)
}

func (s *Server) Login(c echo.Context) error {
	var req api.LoginRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByLogin(ctxOf(c), req.Username)
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return err
		}
		return unauthorized("Invalid credentials")
	}
	return s.issueTokens(c, user)
}

func (s *Server) Refresh(c echo.Context, params api.RefreshParams) error {
	if params.RefreshToken == nil || *params.RefreshToken == "" {
		return unauthorized("No refresh token")
	}
	claims, err := s.tokens.Parse(*params.RefreshToken, auth.TypeRefresh)
	if err != nil {
		return unauthorized("Invalid refresh token")
	}
	user, err := s.store.UserByID(ctxOf(c), claims.Subject)
	if err != nil || claims.TokenVersion < user.TokenVersion {
		return unauthorized("Invalid refresh token")
	}
	access, err := s.tokens.Access(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.LoginResponse{AccessToken: access})
}

func (s *Server) Logout(c echo.Context) error {
	c.SetCookie(s.cookie(c, "", 0))
	return c.NoContent(http.StatusNoContent)
}

// LogoutAll revokes every token of the user on every device.
func (s *Server) LogoutAll(c echo.Context) error {
	if err := s.store.BumpTokenVersion(ctxOf(c), currentUser(c)); err != nil {
		return err
	}
	c.SetCookie(s.cookie(c, "", 0))
	return c.NoContent(http.StatusNoContent)
}

// ChangePassword signs out all other devices and hands this one new tokens.
func (s *Server) ChangePassword(c echo.Context) error {
	var req api.ChangePasswordRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	// 400, not 401: the client would treat a 401 as an expired session
	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		return badRequest("The current password is wrong")
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return badRequest("%s", err.Error())
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	if err := s.store.SetPassword(ctxOf(c), user.ID, hash); err != nil {
		return err
	}
	if user, err = s.store.UserByID(ctxOf(c), user.ID); err != nil {
		return err
	}
	return s.issueTokens(c, user)
}

// --- Password reset links (no email needed: instance admins hand them over) ---

func (s *Server) GetPasswordReset(c echo.Context, token string) error {
	user, err := s.userForReset(c, token)
	if err != nil {
		return c.JSON(http.StatusOK, api.PasswordResetPreviewDto{Valid: false})
	}
	return c.JSON(http.StatusOK, api.PasswordResetPreviewDto{Valid: true, Username: &user.Username})
}

// ResetPassword sets the new password, uses up the link and signs the user out everywhere.
func (s *Server) ResetPassword(c echo.Context, token string) error {
	var req api.ResetPasswordRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.userForReset(c, token)
	if err != nil {
		return badRequest("This reset link is invalid, expired or already used")
	}
	if err := auth.ValidatePassword(req.NewPassword); err != nil {
		return badRequest("%s", err.Error())
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if err := tx.SetPassword(ctxOf(c), user.ID, hash); err != nil {
			return err
		}
		return tx.DeletePasswordReset(ctxOf(c), hashToken(token))
	})
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) userForReset(c echo.Context, token string) (*store.User, error) {
	reset, err := s.store.PasswordResetByHash(ctxOf(c), hashToken(token))
	if err != nil {
		return nil, err
	}
	if reset.ExpiresAt.Before(time.Now()) {
		return nil, store.ErrNotFound
	}
	return s.store.UserByID(ctxOf(c), reset.UserID)
}

func (s *Server) CreatePasswordReset(c echo.Context, userID api.ID) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	if _, err := s.store.UserByID(ctxOf(c), userID); err != nil {
		return orNotFound(err, "User not found: %s", userID)
	}
	token := randomToken()
	adminID := currentUser(c)
	expiresAt := time.Now().UTC().Add(passwordResetValidity)
	if err := s.store.ReplacePasswordReset(ctxOf(c), store.PasswordReset{
		TokenHash: hashToken(token), UserID: userID, CreatedBy: &adminID, ExpiresAt: expiresAt,
	}); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.PasswordResetDto{Token: token, ExpiresAt: expiresAt})
}

// GetAllUsers is for instance admins (ADMIN_USERS), e.g. to create reset links.
func (s *Server) GetAllUsers(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	users, err := s.store.AllUsers(ctxOf(c))
	if err != nil {
		return err
	}
	result := make([]api.AdminUserDto, 0, len(users))
	for _, user := range users {
		createdAt, spaces := user.CreatedAt, user.Spaces
		result = append(result, api.AdminUserDto{
			Id: user.ID, Username: user.Username, Email: user.Email, AvatarId: user.AvatarImageID,
			InstanceAdmin: s.isAdmin(&user.User), CreatedAt: &createdAt, Spaces: &spaces,
		})
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) requireInstanceAdmin(c echo.Context) error {
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	if !s.isAdmin(user) {
		return forbidden("Only instance admins (ADMIN_USERS) may do this")
	}
	return nil
}

func randomToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// hashToken: reset links are stored as hashes, so a leaked database contains no usable links.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
