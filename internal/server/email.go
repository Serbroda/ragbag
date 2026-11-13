package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/mail"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// With SMTP configured (SMTP_HOST), new accounts confirm their email address before they can sign in, and
// everybody can request a password reset link by email. Without it everything works as before: accounts are
// confirmed right away and only instance admins create reset links.

const (
	verificationValidity = 48 * time.Hour
	// unverifiedRetention is how long an account may wait for its confirmation before it is deleted
	unverifiedRetention = 7 * 24 * time.Hour
	emailResetValidity  = time.Hour
)

// UseMailer replaces the SMTP sender (tests); nil turns email off.
func (s *Server) UseMailer(sender mail.Sender) { s.mailer = sender }

func (s *Server) mailEnabled() bool { return s.mailer != nil }

// sendMail delivers in the background: requests answer at once and take the same time whether a mail is sent
// or not (nobody learns which accounts exist). Failures are logged without the message, which holds a link.
func (s *Server) sendMail(user *store.User, subject, text string) {
	s.deliver(user.Email, user.Username, subject, text)
}

// deliver sends to any address; who names the recipient in the log (never the address).
func (s *Server) deliver(to, who, subject, text string) {
	sender := s.mailer
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := sender.Send(ctx, mail.Message{To: to, Subject: subject, Text: text}); err != nil {
			slog.Error("sending an email failed", "user", who, "subject", subject, "error", err)
			return
		}
		slog.Info("email sent", "user", who, "subject", subject)
	}()
}

// mailAllowed keeps anybody from filling an inbox with confirmation or reset mails.
func (s *Server) mailAllowed(user *store.User) bool {
	return s.mails.take(user.ID) == 0
}

func (s *Server) sendVerification(c echo.Context, tx *store.Store, user *store.User) error {
	token := randomToken()
	if err := tx.ReplaceEmailVerification(ctxOf(c), user.ID, hashToken(token), time.Now().Add(verificationValidity)); err != nil {
		return err
	}
	s.sendMail(user, "Confirm your Ragbag account", fmt.Sprintf(`Hi %s,

please confirm your email address to start using Ragbag:

%s/app/verify-email/%s

The link is valid for 48 hours. Accounts that are not confirmed within 7 days are deleted.

If you did not create this account, just ignore this email.
`, user.Name(), s.cfg.PublicURL, token))
	return nil
}

// VerifyEmail confirms the account, creates its first space and signs it in.
func (s *Server) VerifyEmail(c echo.Context, token string) error {
	invalid := badRequest("This confirmation link is invalid or expired. Sign in to get a new one.")
	verification, err := s.store.EmailVerificationByHash(ctxOf(c), hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return invalid
	}
	if err != nil {
		return err
	}
	if verification.ExpiresAt.Before(time.Now()) {
		return invalid
	}
	var user *store.User
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if user, err = tx.UserByID(ctxOf(c), verification.UserID); err != nil {
			return err
		}
		return s.confirmUser(c, tx, user)
	})
	if err != nil {
		return err
	}
	c.Set(usernameKey, user.Username)
	audit(c, slog.LevelInfo, "email address confirmed")
	return s.issueTokens(c, user)
}

// confirmUser marks the address as confirmed and creates the first space, which waits for the confirmation, like
// joining the space of the invite the account registered with.
func (s *Server) confirmUser(c echo.Context, tx *store.Store, user *store.User) error {
	inviteID, err := tx.VerificationInvite(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	if err := tx.SetEmailVerified(ctxOf(c), user.ID); err != nil {
		return err
	}
	if err := createFirstSpace(c, tx, user); err != nil || inviteID == "" {
		return err
	}
	// Revoked meanwhile (or its space deleted): nothing to join
	invite, err := tx.InviteByID(ctxOf(c), inviteID)
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return joinSpace(ctxOf(c), tx, invite.TargetID, user.ID, Role(invite.Role))
}

// RequestPasswordReset mails a reset link; the answer is the same whether the account exists or not.
func (s *Server) RequestPasswordReset(c echo.Context) error {
	if !s.mailEnabled() {
		return badRequest("This server sends no emails; ask an administrator for a reset link")
	}
	var req api.RequestPasswordResetRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByLogin(ctxOf(c), req.Login)
	if errors.Is(err, store.ErrNotFound) {
		audit(c, slog.LevelInfo, "password reset requested for an unknown account")
		return c.NoContent(http.StatusNoContent)
	}
	if err != nil {
		return err
	}
	// Unconfirmed accounts get a new confirmation link when signing in instead
	if user.EmailVerifiedAt == nil || user.BlockedAt != nil || user.DeletedAt != nil || !s.mailAllowed(user) {
		return c.NoContent(http.StatusNoContent)
	}
	token := randomToken()
	if err := s.store.ReplacePasswordReset(ctxOf(c), store.PasswordReset{
		TokenHash: hashToken(token), UserID: user.ID, ExpiresAt: time.Now().UTC().Add(emailResetValidity),
	}); err != nil {
		return err
	}
	s.sendMail(user, "Reset your Ragbag password", fmt.Sprintf(`Hi %s,

somebody (hopefully you) asked to reset the password of your Ragbag account. Choose a new one here:

%s/app/reset-password/%s

The link is valid for one hour and works once. Setting a new password signs you out on all devices.

If you did not ask for this, just ignore this email; your password stays the same.
`, user.Name(), s.cfg.PublicURL, token))
	audit(c, slog.LevelInfo, "password reset link sent by email", "for_user", user.ID)
	return c.NoContent(http.StatusNoContent)
}

// DeleteUnverifiedAccounts removes accounts that were not confirmed within unverifiedRetention (hourly job).
func (s *Server) DeleteUnverifiedAccounts(ctx context.Context) {
	if !s.mailEnabled() {
		return
	}
	deleted, err := s.store.DeleteUnverifiedUsers(ctx, time.Now().Add(-unverifiedRetention))
	if err != nil {
		slog.Error("deleting unconfirmed accounts failed", "error", err)
		return
	}
	if deleted > 0 {
		slog.Info("deleted unconfirmed accounts", "accounts", deleted)
	}
}

// SendTestMail checks the SMTP settings: it sends to the admin's own address while the request waits, so a
// failure shows up in the administration instead of only in the log.
func (s *Server) SendTestMail(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	if !s.mailEnabled() {
		return badRequest("Email is off: set SMTP_HOST and restart")
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	if user.Email == "" {
		return badRequest("Your account has no email address to send to")
	}
	ctx, cancel := context.WithTimeout(ctxOf(c), 30*time.Second)
	defer cancel()
	err = s.mailer.Send(ctx, mail.Message{To: user.Email, Subject: "Ragbag test email", Text: fmt.Sprintf(`Hi %s,

this is a test email from the administration of %s. Email works.
`, user.Name(), s.cfg.PublicURL)})
	if err != nil {
		audit(c, slog.LevelWarn, "test email failed", "error", err)
		return &apiError{http.StatusBadGateway, "Email failed", "The mail server did not take the email: " + err.Error()}
	}
	audit(c, slog.LevelInfo, "test email sent")
	return c.NoContent(http.StatusNoContent)
}
