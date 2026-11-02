package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"log/slog"
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
	flags := map[string]string{
		"registration_enabled": strconv.FormatBool(s.cfg.RegistrationOpen),
		// Accounts confirm their address and passwords can be reset by email
		"email":   strconv.FormatBool(s.mailEnabled()),
		"version": s.version,
	}
	// legal_imprint, legal_privacy, legal_terms: where the pages are, when there are any
	for page := range legalTitles {
		if link := s.legalLink(page); link != "" {
			flags["legal_"+page] = link
		}
	}
	return c.JSON(http.StatusOK, flags)
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
	terms := s.legalLink("terms") != ""
	if terms && (req.AcceptTerms == nil || !*req.AcceptTerms) {
		return badRequest("Please accept the terms of use")
	}
	user, err := s.createUser(c, req.Username, req.Email, req.Password, !s.mailEnabled())
	if err != nil {
		return err
	}
	if terms {
		if err := s.store.AcceptTerms(ctxOf(c), user.ID); err != nil {
			return err
		}
	}
	c.Set(usernameKey, user.Username)
	audit(c, slog.LevelInfo, "account registered", "confirmed", user.EmailVerifiedAt != nil)
	return c.JSON(http.StatusOK, s.userDto(user))
}

// createUser registers an account with its own first space, like Discord gives you a server. Unconfirmed ones
// get a confirmation link instead and their space with the confirmation.
func (s *Server) createUser(c echo.Context, username, email, password string, confirmed bool) (*store.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	email = strings.ToLower(strings.TrimSpace(email))
	if username == "" || len(username) > 80 || strings.ContainsFunc(username, unicode.IsSpace) {
		return nil, badRequest("Usernames must have 1 to 80 characters without spaces")
	}
	// The names deleted accounts leave behind
	if strings.HasPrefix(username, "deleted-") {
		return nil, badRequest("Usernames cannot start with \"deleted-\"")
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
		if user, err = tx.CreateUser(ctxOf(c), username, email, hash, confirmed); err != nil {
			return err
		}
		if !confirmed {
			return s.sendVerification(c, tx, user)
		}
		return createFirstSpace(c, tx, user)
	})
	return user, err
}

func createFirstSpace(c echo.Context, tx *store.Store, user *store.User) error {
	space, err := tx.CreateSpace(ctxOf(c), capitalize(user.Username)+"'s Space", nil, user.ID)
	if err != nil {
		return err
	}
	return tx.AddMember(ctxOf(c), space.ID, user.ID, string(RoleOwner))
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
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	// Failures count per account (username and email are the same one), unknown names by themselves
	account := strings.ToLower(strings.TrimSpace(req.Username))
	if err == nil {
		account = user.ID
	}
	if err := s.loginBlocked(c, account); err != nil {
		audit(c, slog.LevelWarn, "login refused after too many failures", "login", req.Username)
		return err
	}
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) {
		s.loginFailed(account)
		// WARN with the IP: what fail2ban and the like look for
		audit(c, slog.LevelWarn, "login failed", "login", req.Username)
		return unauthorized("Invalid credentials")
	}
	c.Set(usernameKey, user.Username)
	if user.BlockedAt != nil {
		audit(c, slog.LevelWarn, "login of a blocked account refused")
		return blockedMessage(user)
	}
	if user.EmailVerifiedAt == nil {
		if s.mailEnabled() {
			// The password was right, so a new link is no help for strangers
			if s.mailAllowed(user) {
				if err := s.sendVerification(c, s.store, user); err != nil {
					return err
				}
			}
			audit(c, slog.LevelInfo, "login refused until the email address is confirmed")
			return forbidden("Please confirm your email address first: we sent a new link to %s", user.Email)
		}
		// Email was turned off since the registration: nothing left to wait for
		if err := s.store.InTx(ctxOf(c), func(tx *store.Store) error { return s.confirmUser(c, tx, user) }); err != nil {
			return err
		}
	}
	audit(c, slog.LevelInfo, "login")
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
	if err != nil || claims.TokenVersion < user.TokenVersion || user.BlockedAt != nil || user.DeletedAt != nil {
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
	audit(c, slog.LevelInfo, "logged out on all devices")
	c.SetCookie(s.cookie(c, "", 0))
	return c.NoContent(http.StatusNoContent)
}

// ChangePassword signs out all other devices and hands this one new tokens.
func (s *Server) ChangePassword(c echo.Context) error {
	if err := s.limitPasswordChange(c); err != nil {
		return err
	}
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
	if user.MustChangePassword {
		s.removeInitialPassword()
	}
	if user, err = s.store.UserByID(ctxOf(c), user.ID); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "password changed")
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
	audit(c, slog.LevelInfo, "password reset with a reset link")
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
	if user, err := s.store.UserByID(ctxOf(c), userID); err != nil || user.DeletedAt != nil {
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
	audit(c, slog.LevelInfo, "password reset link created", "for_user", userID)
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
		result = append(result, s.adminUserDto(user))
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
