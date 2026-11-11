package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Two-factor authentication (TOTP), off unless a user turns it on in their profile: the login then also needs
// the code of their authenticator app or one of the recovery codes, each of which works once. Password reset
// links do not sign anybody in, so they never get around it; who lost app and recovery codes needs an instance
// admin (or `ragbag reset-2fa` on the server). The secret is encrypted with a key derived from JWT_SECRET: with
// another JWT_SECRET (also after restoring a backup elsewhere) only the recovery codes still work.

const (
	// twoFactorHeader tells the login page to ask for the code: the password was right, the code is missing
	twoFactorHeader   = "X-Ragbag-Two-Factor"
	twoFactorIssuer   = "Ragbag"
	recoveryCodeCount = 10
	// After maxCodeFailures wrong codes in a row checking pauses for codeBlockDuration
	maxCodeFailures   = 5
	codeBlockDuration = 5 * time.Minute
)

// secondFactor checks a code of the app or a recovery code and remembers the outcome: a code is used up, wrong
// ones count. With appOnly recovery codes are not taken (confirming the app while setting it up).
func (s *Server) secondFactor(c echo.Context, user *store.User, state *store.TwoFactor, code string, appOnly bool) error {
	if state.BlockedUntil != nil && time.Now().Before(*state.BlockedUntil) {
		return tooManyRequests("Too many wrong codes, try again in a few minutes")
	}
	if state.Secret != nil {
		// Fails with another JWT_SECRET than the one it was set up with; recovery codes still work then
		if secret, err := s.secrets.Open(*state.Secret); err == nil {
			if step, ok := auth.CheckTOTP(secret, code, time.Now(), state.LastStep); ok {
				return s.store.TwoFactorPassed(ctxOf(c), user.ID, step, state.Recovery)
			}
		} else {
			slog.Warn("two-factor secret cannot be read, was JWT_SECRET changed? Only recovery codes work", "user", user.Username)
		}
	}
	if !appOnly {
		hashes := recoveryHashes(state)
		if at := slices.Index(hashes, hashToken(auth.NormalizeRecoveryCode(code))); at >= 0 {
			left, _ := json.Marshal(slices.Delete(hashes, at, at+1))
			audit(c, slog.LevelWarn, "signed in with a recovery code", "left", len(hashes)-1)
			return s.store.TwoFactorPassed(ctxOf(c), user.ID, state.LastStep, ptr(string(left)))
		}
	}
	failures, until := state.Failures+1, (*time.Time)(nil)
	if failures >= maxCodeFailures {
		blocked := time.Now().Add(codeBlockDuration)
		failures, until = 0, &blocked
	}
	if err := s.store.SetTwoFactorFailures(ctxOf(c), user.ID, failures, until); err != nil {
		return err
	}
	if until != nil {
		audit(c, slog.LevelWarn, "two-factor codes paused after too many wrong ones")
		return tooManyRequests("Too many wrong codes, try again in a few minutes")
	}
	audit(c, slog.LevelWarn, "wrong two-factor code")
	return badRequest("Wrong code")
}

func recoveryHashes(state *store.TwoFactor) []string {
	var hashes []string
	if state.Recovery != nil {
		_ = json.Unmarshal([]byte(*state.Recovery), &hashes)
	}
	return hashes
}

// loginSecondFactor is the second step of a login once the password was right; accounts without it pass.
func (s *Server) loginSecondFactor(c echo.Context, user *store.User, code *string, account string) error {
	state, err := s.store.TwoFactorState(ctxOf(c), user.ID)
	if err != nil || state.EnabledAt == nil {
		return err
	}
	if code == nil || strings.TrimSpace(*code) == "" {
		c.Response().Header().Set(twoFactorHeader, "required")
		return unauthorized("Enter the code of your authenticator app")
	}
	err = s.secondFactor(c, user, state, *code, false)
	if apiErr, ok := err.(*apiError); ok {
		s.loginFailed(account)
		if apiErr.status == http.StatusBadRequest {
			// Still the same step for the login page
			c.Response().Header().Set(twoFactorHeader, "required")
			return unauthorized("Wrong code")
		}
	}
	return err
}

// SetupTwoFactor creates a secret for the authenticator app. Nothing changes for the login until the first code
// of the app confirms it (EnableTwoFactor); starting again replaces a secret that was never confirmed.
func (s *Server) SetupTwoFactor(c echo.Context) error {
	var req api.TwoFactorSetupRequest
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
	state, err := s.store.TwoFactorState(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	if state.EnabledAt != nil {
		return conflict("Two-factor authentication is already on")
	}
	secret := auth.NewTOTPSecret()
	if err := s.store.StartTwoFactor(ctxOf(c), user.ID, s.secrets.Seal(secret)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.TwoFactorSetupDto{Secret: secret, Uri: auth.TOTPURI(twoFactorIssuer, user.Username, secret)})
}

// EnableTwoFactor turns it on with the first code of the app and returns the recovery codes, shown only now.
func (s *Server) EnableTwoFactor(c echo.Context) error {
	var req api.TwoFactorCodeRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	state, err := s.store.TwoFactorState(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	if state.EnabledAt != nil {
		return conflict("Two-factor authentication is already on")
	}
	if state.Secret == nil {
		return badRequest("Start the setup first")
	}
	if err := s.secondFactor(c, user, state, req.Code, true); err != nil {
		return err
	}
	codes, hashes := make([]string, recoveryCodeCount), make([]string, recoveryCodeCount)
	for i := range codes {
		codes[i] = auth.NewRecoveryCode()
		hashes[i] = hashToken(auth.NormalizeRecoveryCode(codes[i]))
	}
	stored, _ := json.Marshal(hashes)
	if err := s.store.EnableTwoFactor(ctxOf(c), user.ID, string(stored)); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "two-factor authentication turned on")
	return c.JSON(http.StatusOK, api.TwoFactorRecoveryDto{RecoveryCodes: codes})
}

// DisableTwoFactor needs the password and a code (of the app or a recovery code), like a login.
func (s *Server) DisableTwoFactor(c echo.Context) error {
	var req api.TwoFactorDisableRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	if !auth.CheckPassword(user.PasswordHash, req.CurrentPassword) {
		return badRequest("The current password is wrong")
	}
	state, err := s.store.TwoFactorState(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	if state.EnabledAt != nil {
		if err := s.secondFactor(c, user, state, req.Code, false); err != nil {
			return err
		}
	}
	if err := s.store.ClearTwoFactor(ctxOf(c), user.ID); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "two-factor authentication turned off")
	return c.NoContent(http.StatusNoContent)
}

// ResetTwoFactor is for instance admins: somebody lost their app and their recovery codes.
func (s *Server) ResetTwoFactor(c echo.Context, userID api.ID) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	if user, err := s.store.UserByID(ctxOf(c), userID); err != nil || user.DeletedAt != nil {
		return orNotFound(err, "User not found: %s", userID)
	}
	if err := s.store.ClearTwoFactor(ctxOf(c), userID); err != nil {
		return err
	}
	audit(c, slog.LevelWarn, "two-factor authentication reset by an admin", "for_user", userID)
	return c.NoContent(http.StatusNoContent)
}
