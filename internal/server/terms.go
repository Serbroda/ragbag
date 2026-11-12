package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// When the terms of use change (instance admins, /admin), every account has to accept them again from the day
// they apply: until then everything but its own account answers 403 with termsHeader, API tokens included. A
// change can be announced ahead (the app shows a notice), new accounts accept the terms at registration.

const termsHeader = "X-Ragbag-Terms"

// termsPending: the terms changed since the user last accepted them (or they never did).
func (s *Server) termsPending(c echo.Context, user *store.User) (bool, error) {
	since, err := s.store.TermsInForceSince(ctxOf(c), time.Now())
	if err != nil || since == nil {
		return false, err
	}
	if user.TermsAcceptedAt != nil && !user.TermsAcceptedAt.Before(*since) {
		return false, nil
	}
	// Without terms (any more) there is nothing to accept
	return s.legalLink("terms") != "", nil
}

// requireTerms answers 403 while the user has to accept changed terms, except for what deciding needs.
func (s *Server) requireTerms(c echo.Context, user *store.User) error {
	if allowedPendingTerms(c.Request().Method, c.Request().URL.Path) {
		return nil
	}
	pending, err := s.termsPending(c, user)
	if err != nil {
		return err
	}
	if pending {
		// The frontend opens the page to accept them from wherever it was
		c.Response().Header().Set(termsHeader, "accept")
		return forbidden("The terms of use have changed: accept them in Ragbag to go on")
	}
	return nil
}

// allowedPendingTerms: reading the own account, accepting and deleting it (who does not agree may leave).
func allowedPendingTerms(method, requestPath string) bool {
	switch {
	case method == http.MethodGet && requestPath == "/api/v1/users/me":
		return true
	case method == http.MethodPost && (requestPath == "/api/v1/users/me/terms" || requestPath == "/api/v1/users/me/delete"):
		return true
	}
	return false
}

func (s *Server) AcceptTerms(c echo.Context) error {
	if s.legalLink("terms") == "" {
		return badRequest("This Ragbag has no terms of use")
	}
	if err := s.store.AcceptTerms(ctxOf(c), currentUser(c)); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "terms accepted")
	return c.NoContent(http.StatusNoContent)
}

// termsOfMe adds the terms to the current user: whether they have to accept them, and an announced change.
func (s *Server) termsOfMe(c echo.Context, user *store.User, dto *api.UserDto) error {
	pending, err := s.termsPending(c, user)
	if err != nil {
		return err
	}
	if pending {
		dto.TermsPending = &pending
	}
	if s.legalLink("terms") == "" {
		return nil
	}
	next, err := s.store.NextTermsChange(ctxOf(c), time.Now())
	dto.TermsChangeAt = next
	return err
}

func (s *Server) GetTermsState(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	return s.termsState(c)
}

func (s *Server) ChangeTerms(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	var req api.ChangeTermsRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	if s.legalLink("terms") == "" {
		return badRequest("This Ragbag has no terms of use: put them into DATA_DIR/legal/terms.md or set TERMS_URL")
	}
	effectiveAt := time.Now()
	if req.EffectiveAt != nil && req.EffectiveAt.After(effectiveAt) {
		effectiveAt = *req.EffectiveAt
	}
	if err := s.store.AddTermsChange(ctxOf(c), effectiveAt, currentUser(c)); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "terms changed", "effective_at", effectiveAt.UTC().Format(time.RFC3339))
	return s.termsState(c)
}

func (s *Server) WithdrawTermsChange(c echo.Context) error {
	if err := s.requireInstanceAdmin(c); err != nil {
		return err
	}
	withdrawn, err := s.store.DeleteTermsChangesAfter(ctxOf(c), time.Now())
	if err != nil {
		return err
	}
	if !withdrawn {
		return notFound("No change of the terms of use is announced")
	}
	audit(c, slog.LevelInfo, "terms change withdrawn")
	return s.termsState(c)
}

func (s *Server) termsState(c echo.Context) error {
	at := time.Now()
	since, err := s.store.TermsInForceSince(ctxOf(c), at)
	if err != nil {
		return err
	}
	next, err := s.store.NextTermsChange(ctxOf(c), at)
	if err != nil {
		return err
	}
	// Before the first change, every acceptance counts
	var from time.Time
	if since != nil {
		from = *since
	}
	accepted, accounts, err := s.store.CountTermsAccepted(ctxOf(c), from)
	if err != nil {
		return err
	}
	state := api.TermsStateDto{InForceSince: since, Next: next, Accepted: accepted, Accounts: accounts}
	if link := s.legalLink("terms"); link != "" {
		state.Terms = &link
	}
	return c.JSON(http.StatusOK, state)
}
