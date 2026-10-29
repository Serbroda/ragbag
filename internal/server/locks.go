package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Locked collections: each user can hide collections from their own view (against looks over the shoulder).
// They are part of hiddenCollections until the user unlocks with their PIN, so every list, search, dashboard,
// counter and export leaves them out. Unlocking gives a short-lived token for the X-Ragbag-Unlock header; the
// frontend renews it while the user is active and drops it to lock again.

const (
	unlockHeader = "X-Ragbag-Unlock"
	unlockTTL    = 10 * time.Minute
	// After maxPinFailures wrong PINs in a row unlocking pauses for pinBlockDuration
	maxPinFailures   = 5
	pinBlockDuration = 5 * time.Minute
)

type unlockedKey struct{}

// withUnlock marks the request as unlocked if it carries a valid unlock token of the signed-in user.
func (s *Server) withUnlock(c echo.Context, user *store.User) {
	token := c.Request().Header.Get(unlockHeader)
	if token == "" {
		return
	}
	claims, err := s.tokens.Parse(token, auth.TypeUnlock)
	if err != nil || claims.Subject != user.ID || claims.TokenVersion < user.TokenVersion {
		return
	}
	c.SetRequest(c.Request().WithContext(context.WithValue(c.Request().Context(), unlockedKey{}, true)))
}

func isUnlocked(ctx context.Context) bool {
	unlocked, _ := ctx.Value(unlockedKey{}).(bool)
	return unlocked
}

func tooManyRequests(format string, args ...any) error {
	err := badRequest(format, args...).(*apiError)
	err.status, err.title = http.StatusTooManyRequests, "Too many requests"
	return err
}

func (s *Server) SetPin(c echo.Context) error {
	var req api.SetPinDto
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
	if err := auth.ValidatePIN(req.Pin); err != nil {
		return badRequest("%s", err.Error())
	}
	hash, err := auth.HashPassword(req.Pin)
	if err != nil {
		return err
	}
	if err := s.store.SetPin(ctxOf(c), user.ID, hash); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) Unlock(c echo.Context) error {
	var req api.UnlockRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	state, err := s.store.PinState(ctxOf(c), user.ID)
	if err != nil {
		return err
	}
	if state.Hash == nil {
		return badRequest("Set a PIN in your profile first")
	}
	if state.BlockedUntil != nil && time.Now().Before(*state.BlockedUntil) {
		return tooManyRequests("Too many wrong PINs, try again in a few minutes")
	}
	if !auth.CheckPassword(*state.Hash, req.Pin) {
		failures, until := state.Failures+1, (*time.Time)(nil)
		if failures >= maxPinFailures {
			blocked := time.Now().Add(pinBlockDuration)
			failures, until = 0, &blocked
		}
		if err := s.store.SetPinFailures(ctxOf(c), user.ID, failures, until); err != nil {
			return err
		}
		if until != nil {
			audit(c, slog.LevelWarn, "unlocking paused after too many wrong PINs")
			return tooManyRequests("Too many wrong PINs, try again in a few minutes")
		}
		audit(c, slog.LevelWarn, "wrong PIN")
		return badRequest("Wrong PIN")
	}
	if state.Failures > 0 || state.BlockedUntil != nil {
		if err := s.store.SetPinFailures(ctxOf(c), user.ID, 0, nil); err != nil {
			return err
		}
	}
	return s.issueUnlock(c, user)
}

// RenewUnlock extends a valid unlock, so active users stay unlocked and idle ones are locked after unlockTTL.
func (s *Server) RenewUnlock(c echo.Context) error {
	if !isUnlocked(ctxOf(c)) {
		return badRequest("Locked collections are locked again")
	}
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	return s.issueUnlock(c, user)
}

func (s *Server) issueUnlock(c echo.Context, user *store.User) error {
	token, expiresAt, err := s.tokens.Unlock(user.ID, user.TokenVersion, unlockTTL)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.UnlockDto{Token: token, ExpiresAt: expiresAt})
}

// LockCollection hides the collection from the current user's view; nobody else is affected.
func (s *Server) LockCollection(c echo.Context, collectionID api.ID) error {
	if _, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead); err != nil {
		return err
	}
	state, err := s.store.PinState(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	// Without a PIN the collection could never be seen again
	if state.Hash == nil {
		return badRequest("Set a PIN in your profile first")
	}
	if err := s.store.LockCollection(ctxOf(c), currentUser(c), collectionID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// UnlockCollection removes the lock; a locked collection is only reachable (and so unlockable) while unlocked.
func (s *Server) UnlockCollection(c echo.Context, collectionID api.ID) error {
	if _, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead); err != nil {
		return err
	}
	if err := s.store.UnlockCollection(ctxOf(c), currentUser(c), collectionID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// lockedCollections are the collections of the space the user locked, each with everything below it.
func lockedCollections(collections []store.Collection, locks []string) map[string]bool {
	locked := map[string]bool{}
	if len(locks) == 0 {
		return locked
	}
	children := map[string][]string{}
	for _, collection := range collections {
		if collection.ParentID != nil {
			children[*collection.ParentID] = append(children[*collection.ParentID], collection.ID)
		}
	}
	queue := append([]string{}, locks...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if locked[id] {
			continue
		}
		locked[id] = true
		queue = append(queue, children[id]...)
	}
	return locked
}
