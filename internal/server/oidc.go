package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
)

// Single sign-on with an OpenID Connect provider (OIDC_ISSUER): the authorization code flow with PKCE and a
// nonce. Accounts are connected by the provider's issuer and subject (user_identities), never by email: who can
// set any address at the provider would otherwise take over accounts here. People the provider knows without an
// account get a ticket to choose their username (OIDC_AUTO_REGISTER, or an invite that registers). Signing in
// this way skips Ragbag's own two-factor authentication, the provider has its own.

const (
	oidcCookie     = "oidc_state"
	oidcCookiePath = "/api/auth/oidc"
	// oidcFlowTTL is how long the provider's sign-in page may take
	oidcFlowTTL = 10 * time.Minute
	// oidcTicketTTL is how long choosing a username may take
	oidcTicketTTL = 15 * time.Minute
	// ssoSignupKey marks a sign-up by single sign-on: with OIDC_AUTO_REGISTER every invite lets them in
	ssoSignupKey = "ssoSignup"
)

// oidcFlow is a sign-in started in this browser (by the state in its cookie).
type oidcFlow struct {
	verifier string
	nonce    string
	// userID is the account being connected, empty when signing in
	userID  string
	expires time.Time
}

// oidcPerson is who the provider says somebody is.
type oidcPerson struct {
	issuer, subject   string
	email             string
	emailVerified     bool
	preferredUsername string
	name              string
	expires           time.Time
}

type oidcClient struct {
	cfg         config.OIDC
	redirectURL string
	httpClient  *http.Client

	mu       sync.Mutex
	provider *oidc.Provider
	flows    map[string]oidcFlow
	// tickets of people without an account, until they chose a username
	tickets map[string]oidcPerson
}

func newOIDCClient(cfg config.OIDC, publicURL string) *oidcClient {
	return &oidcClient{
		cfg: cfg, redirectURL: publicURL + "/api/auth/oidc/callback",
		// The provider is chosen by the admin and often lives in the own network: no SSRF guard here
		httpClient: &http.Client{Timeout: 15 * time.Second},
		flows:      map[string]oidcFlow{}, tickets: map[string]oidcPerson{},
	}
}

// discover reads the provider's configuration on first use (and again after a failure), so a provider that is
// down does not keep Ragbag from starting.
func (o *oidcClient) discover(ctx context.Context) (*oidc.Provider, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.provider != nil {
		return o.provider, nil
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, o.httpClient), o.cfg.Issuer)
	if err != nil {
		return nil, err
	}
	o.provider = provider
	return provider, nil
}

func (o *oidcClient) oauth(provider *oidc.Provider) *oauth2.Config {
	return &oauth2.Config{ClientID: o.cfg.ClientID, ClientSecret: o.cfg.ClientSecret, Endpoint: provider.Endpoint(),
		RedirectURL: o.redirectURL, Scopes: []string{oidc.ScopeOpenID, "profile", "email"}}
}

// start remembers a new flow and returns the provider's sign-in address and the state for the cookie.
func (o *oidcClient) start(ctx context.Context, userID string) (string, string, error) {
	provider, err := o.discover(ctx)
	if err != nil {
		return "", "", err
	}
	state, flow := randomToken(), oidcFlow{verifier: oauth2.GenerateVerifier(), nonce: randomToken(), userID: userID,
		expires: time.Now().Add(oidcFlowTTL)}
	o.mu.Lock()
	for key, old := range o.flows {
		if time.Now().After(old.expires) {
			delete(o.flows, key)
		}
	}
	o.flows[state] = flow
	o.mu.Unlock()
	address := o.oauth(provider).AuthCodeURL(state, oidc.Nonce(flow.nonce), oauth2.S256ChallengeOption(flow.verifier))
	return address, state, nil
}

// take ends the flow of the state, which has to be the one in this browser's cookie (no login CSRF).
func (o *oidcClient) take(state, cookie string) (oidcFlow, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	flow, ok := o.flows[state]
	delete(o.flows, state)
	return flow, ok && state != "" && state == cookie && time.Now().Before(flow.expires)
}

// person exchanges the code and checks the ID token: signature, issuer, audience, expiry and nonce.
func (o *oidcClient) person(ctx context.Context, code string, flow oidcFlow) (*oidcPerson, error) {
	provider, err := o.discover(ctx)
	if err != nil {
		return nil, err
	}
	ctx = oidc.ClientContext(ctx, o.httpClient)
	token, err := o.oauth(provider).Exchange(ctx, code, oauth2.VerifierOption(flow.verifier))
	if err != nil {
		return nil, err
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no ID token")
	}
	idToken, err := provider.Verifier(&oidc.Config{ClientID: o.cfg.ClientID}).Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	if idToken.Nonce != flow.nonce {
		return nil, errors.New("wrong nonce")
	}
	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, err
	}
	// Some providers only tell the address via the user info
	if claims.Email == "" {
		if info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token)); err == nil {
			_ = info.Claims(&claims)
			claims.Email, claims.EmailVerified = info.Email, info.EmailVerified
		}
	}
	return &oidcPerson{issuer: idToken.Issuer, subject: idToken.Subject, email: strings.ToLower(claims.Email),
		emailVerified: claims.EmailVerified, preferredUsername: claims.PreferredUsername, name: claims.Name}, nil
}

func (o *oidcClient) addTicket(person oidcPerson) string {
	ticket := randomToken()
	person.expires = time.Now().Add(oidcTicketTTL)
	o.mu.Lock()
	defer o.mu.Unlock()
	for key, old := range o.tickets {
		if time.Now().After(old.expires) {
			delete(o.tickets, key)
		}
	}
	o.tickets[ticket] = person
	return ticket
}

func (o *oidcClient) ticket(ticket string) (oidcPerson, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	person, ok := o.tickets[ticket]
	return person, ok && time.Now().Before(person.expires)
}

func (o *oidcClient) useTicket(ticket string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.tickets, ticket)
}

// --- Handlers ---

func (s *Server) StartOidcLogin(c echo.Context) error {
	return s.startOidc(c, "")
}

func (s *Server) ConnectOidc(c echo.Context) error {
	if s.sso == nil {
		return notFound("Single sign-on is not configured")
	}
	if _, err := s.store.IdentityOf(ctxOf(c), currentUser(c), s.cfg.OIDC.Issuer); err == nil {
		return conflict("Your account is connected already")
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	return s.startOidc(c, currentUser(c))
}

func (s *Server) startOidc(c echo.Context, userID string) error {
	if s.sso == nil {
		return notFound("Single sign-on is not configured")
	}
	address, state, err := s.sso.start(ctxOf(c), userID)
	if err != nil {
		slog.Error("single sign-on provider not reachable", "issuer", s.cfg.OIDC.Issuer, "error", err)
		return &echo.HTTPError{Code: http.StatusBadGateway, Message: fmt.Sprintf("%s cannot be reached right now", s.cfg.OIDC.Name)}
	}
	c.SetCookie(s.oidcStateCookie(c, state, oidcFlowTTL))
	return c.JSON(http.StatusOK, api.OidcStartDto{Url: address})
}

func (s *Server) oidcStateCookie(c echo.Context, value string, maxAge time.Duration) *http.Cookie {
	cookie := &http.Cookie{Name: oidcCookie, Value: value, Path: oidcCookiePath, HttpOnly: true,
		// Lax: it has to come along when the provider sends the browser back
		SameSite: http.SameSiteLaxMode, Secure: s.cfg.ForceSecureCookie || c.Scheme() == "https", MaxAge: -1}
	if maxAge > 0 {
		cookie.MaxAge = int(maxAge.Seconds())
	}
	return cookie
}

// OidcCallback always answers with a redirect into the app, failures as sso_error (shown there).
func (s *Server) OidcCallback(c echo.Context, params api.OidcCallbackParams) error {
	if s.sso == nil {
		return notFound("Single sign-on is not configured")
	}
	cookie := ""
	if found, err := c.Cookie(oidcCookie); err == nil {
		cookie = found.Value
	}
	c.SetCookie(s.oidcStateCookie(c, "", 0))
	state := ""
	if params.State != nil {
		state = *params.State
	}
	flow, ok := s.sso.take(state, cookie)
	back := "/login"
	if flow.userID != "" {
		back = "/settings/security"
	}
	fail := func(message string) error {
		return c.Redirect(http.StatusFound, back+"?"+url.Values{"sso_error": {message}}.Encode())
	}
	if !ok {
		return fail("The sign-in took too long or was started in another browser. Please try again.")
	}
	if params.Error != nil || params.Code == nil {
		return fail(fmt.Sprintf("The sign-in with %s was cancelled.", s.cfg.OIDC.Name))
	}
	person, err := s.sso.person(ctxOf(c), *params.Code, flow)
	if err != nil {
		audit(c, slog.LevelWarn, "single sign-on failed", "error", err.Error())
		return fail(fmt.Sprintf("%s did not confirm the sign-in. Please try again.", s.cfg.OIDC.Name))
	}
	if flow.userID != "" {
		return s.connectIdentity(c, flow.userID, person, fail)
	}
	return s.signInWithIdentity(c, person, fail)
}

func (s *Server) connectIdentity(c echo.Context, userID string, person *oidcPerson, fail func(string) error) error {
	c.Set(userIDKey, userID)
	identity, err := s.store.IdentityBySubject(ctxOf(c), person.issuer, person.subject)
	switch {
	case err == nil && identity.UserID != userID:
		return fail(fmt.Sprintf("This %s login is connected to another account.", s.cfg.OIDC.Name))
	case err == nil:
		// Connected already
	case errors.Is(err, store.ErrNotFound):
		if err := s.store.AddIdentity(ctxOf(c), store.Identity{Issuer: person.issuer, Subject: person.subject,
			UserID: userID, Email: blankToNil(&person.email)}); err != nil {
			return err
		}
		audit(c, slog.LevelInfo, "account connected to single sign-on")
	default:
		return err
	}
	return c.Redirect(http.StatusFound, "/settings/security?sso=connected")
}

func (s *Server) signInWithIdentity(c echo.Context, person *oidcPerson, fail func(string) error) error {
	identity, err := s.store.IdentityBySubject(ctxOf(c), person.issuer, person.subject)
	if errors.Is(err, store.ErrNotFound) {
		return s.offerSignup(c, person, fail)
	}
	if err != nil {
		return err
	}
	user, err := s.store.UserByID(ctxOf(c), identity.UserID)
	if err != nil || user.DeletedAt != nil {
		return fail("This account does not exist any more.")
	}
	c.Set(usernameKey, user.Username)
	if user.BlockedAt != nil {
		audit(c, slog.LevelWarn, "login of a blocked account refused")
		return fail(blockedMessage(user).Error())
	}
	if user.EmailVerifiedAt == nil {
		if s.mailEnabled() {
			return fail(fmt.Sprintf("Please confirm your email address first: open the link we sent to %s.", user.Email))
		}
		if err := s.store.InTx(ctxOf(c), func(tx *store.Store) error { return s.confirmUser(c, tx, user) }); err != nil {
			return err
		}
	}
	refresh, err := s.tokens.Refresh(user.ID, user.TokenVersion)
	if err != nil {
		return err
	}
	c.SetCookie(s.cookie(c, refresh, s.tokens.RefreshTTL()))
	audit(c, slog.LevelInfo, "login", "via", "single sign-on")
	// The app trades the cookie for an access token there
	return c.Redirect(http.StatusFound, "/login?sso=done")
}

// offerSignup hands people without an account a ticket to choose their username; the app decides whether to
// show it (OIDC_AUTO_REGISTER or an invite), OidcSignup checks it again.
func (s *Server) offerSignup(c echo.Context, person *oidcPerson, fail func(string) error) error {
	if person.email == "" {
		return fail(fmt.Sprintf("%s did not tell your email address, which an account here needs.", s.cfg.OIDC.Name))
	}
	if taken, err := s.store.UsernameOrEmailTaken(ctxOf(c), "", person.email); err != nil {
		return err
	} else if taken {
		return fail(fmt.Sprintf("There is an account with %s already: sign in with its password and connect %s "+
			"under Settings → Security.", person.email, s.cfg.OIDC.Name))
	}
	// In the fragment: never sent to a server, so it ends up in no log
	fragment := url.Values{"ticket": {s.sso.addTicket(*person)}, "email": {person.email},
		"username": {person.preferredUsername}, "name": {person.name}}
	return c.Redirect(http.StatusFound, "/login?sso=signup#"+fragment.Encode())
}

func (s *Server) OidcSignup(c echo.Context) error {
	if s.sso == nil {
		return notFound("Single sign-on is not configured")
	}
	var req api.OidcSignupRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	person, ok := s.sso.ticket(req.Ticket)
	if !ok {
		return badRequest("This sign-up took too long. Please sign in with %s again.", s.cfg.OIDC.Name)
	}
	inviteToken := ""
	if req.InviteToken != nil {
		inviteToken = *req.InviteToken
	}
	if !s.cfg.OIDC.AutoRegister && inviteToken == "" {
		return forbidden("No account is connected to this %s login. Sign in with your password and connect it "+
			"under Settings → Security.", s.cfg.OIDC.Name)
	}
	terms := s.legalLink("terms") != ""
	if terms && (req.AcceptTerms == nil || !*req.AcceptTerms) {
		return badRequest("Please accept the terms of use")
	}
	name, err := displayName(req.DisplayName)
	if err != nil {
		return err
	}
	c.Set(ssoSignupKey, true)
	// The provider vouches for addresses it confirmed; others get the usual confirmation link (if there is mail)
	identity := &store.Identity{Issuer: person.issuer, Subject: person.subject, Email: &person.email}
	user, _, err := s.createUser(c, req.Username, person.email, "", name, inviteToken,
		person.emailVerified || !s.mailEnabled(), identity)
	if err != nil {
		return err
	}
	s.sso.useTicket(req.Ticket)
	if terms {
		if err := s.store.AcceptTerms(ctxOf(c), user.ID); err != nil {
			return err
		}
	}
	c.Set(usernameKey, user.Username)
	audit(c, slog.LevelInfo, "account registered", "via", "single sign-on", "confirmed", user.EmailVerifiedAt != nil)
	if user.EmailVerifiedAt == nil {
		return c.NoContent(http.StatusAccepted)
	}
	return s.issueTokens(c, user)
}

func (s *Server) DisconnectOidc(c echo.Context) error {
	user, err := s.store.UserByID(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	if user.PasswordHash == "" {
		return conflict("Set a password first, otherwise you could not sign in any more")
	}
	if s.cfg.OIDC.Only && !s.isAdmin(user) {
		return conflict("%s is the only way to sign in here", s.cfg.OIDC.Name)
	}
	if err := s.store.RemoveIdentity(ctxOf(c), user.ID, s.cfg.OIDC.Issuer); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "account disconnected from single sign-on")
	return c.NoContent(http.StatusNoContent)
}

// ssoOf tells as which the user is connected to the provider: its email address, else "connected"; nil if not.
func (s *Server) ssoOf(c echo.Context, userID string) (*string, error) {
	if s.sso == nil {
		return nil, nil
	}
	identity, err := s.store.IdentityOf(ctxOf(c), userID, s.cfg.OIDC.Issuer)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if identity.Email != nil {
		return identity.Email, nil
	}
	return ptr("connected"), nil
}

// confirmPassword checks the password asked for before changes to the account. Accounts made by single sign-on
// have none until they set one.
func confirmPassword(user *store.User, password string) error {
	if user.PasswordHash == "" {
		return badRequest("Set a password first (Settings → Security): it confirms changes like this one")
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return badRequest("The current password is wrong")
	}
	return nil
}
