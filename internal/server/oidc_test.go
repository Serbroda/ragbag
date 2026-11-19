package server_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/require"
)

// fakeProvider is an OpenID Connect provider for tests: the test plays the person signing in on its page
// (approve) and the provider answers ragbag's code exchange with a signed ID token.
type fakeProvider struct {
	t      *testing.T
	server *httptest.Server
	key    *rsa.PrivateKey
	mu     sync.Mutex
	codes  map[string]fakeCode
}

type fakeCode struct {
	claims    map[string]any
	nonce     string
	challenge string
}

func newFakeProvider(t *testing.T) *fakeProvider {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	p := &fakeProvider{t: t, key: key, codes: map[string]fakeCode{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"issuer": p.server.URL, "authorization_endpoint": p.server.URL + "/authorize",
			"token_endpoint": p.server.URL + "/token", "jwks_uri": p.server.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"}})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "k", Algorithm: "RS256", Use: "sig"}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		p.mu.Lock()
		code, ok := p.codes[r.Form.Get("code")]
		delete(p.codes, r.Form.Get("code"))
		p.mu.Unlock()
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if !ok || base64.RawURLEncoding.EncodeToString(sum[:]) != code.challenge {
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(w, map[string]string{"error": "invalid_grant"})
			return
		}
		writeJSON(w, map[string]any{"access_token": "at", "token_type": "Bearer", "expires_in": 60,
			"id_token": p.idToken(code.claims, code.nonce)})
	})
	p.server = httptest.NewServer(mux)
	t.Cleanup(p.server.Close)
	return p
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func (p *fakeProvider) idToken(claims map[string]any, nonce string) string {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: p.key},
		(&jose.SignerOptions{}).WithHeader("kid", "k").WithType("JWT"))
	require.NoError(p.t, err)
	all := map[string]any{"iss": p.server.URL, "aud": "ragbag", "nonce": nonce,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix()}
	for key, value := range claims {
		all[key] = value
	}
	token, err := jwt.Signed(signer).Claims(all).Serialize()
	require.NoError(p.t, err)
	return token
}

// approve is the person signing in on the provider's page: it returns the code ragbag gets back.
func (p *fakeProvider) approve(authorizeURL string, claims map[string]any) string {
	parsed, err := url.Parse(authorizeURL)
	require.NoError(p.t, err)
	query := parsed.Query()
	require.Equal(p.t, "S256", query.Get("code_challenge_method"))
	require.Equal(p.t, "ragbag", query.Get("client_id"))
	code := fmt.Sprintf("code-%d", time.Now().UnixNano())
	p.mu.Lock()
	p.codes[code] = fakeCode{claims: claims, nonce: query.Get("nonce"), challenge: query.Get("code_challenge")}
	p.mu.Unlock()
	return code
}

func withOIDC(p *fakeProvider, change func(*config.OIDC)) func(*config.Config) {
	return func(cfg *config.Config) {
		cfg.PublicURL = "http://ragbag.test"
		cfg.OIDC = config.OIDC{Issuer: p.server.URL, ClientID: "ragbag", ClientSecret: "s", Name: "Pocket ID"}
		if change != nil {
			change(&cfg.OIDC)
		}
	}
}

// ssoRound starts a sign-in (or connecting, with a session), lets the provider approve it as the claims and
// returns where the callback sends the browser and the cookies it set.
func ssoRound(e *env, p *fakeProvider, s *session, claims map[string]any) (*url.URL, []*http.Cookie) {
	e.t.Helper()
	start := e.req("POST", "/api/auth/oidc/start")
	if s != nil {
		start = e.req("POST", "/api/v1/users/me/oidc").as(s)
	}
	res := start.expect(200)
	state := res.cookieNamed("oidc_state")
	require.NotNil(e.t, state)
	address := res.str("url")
	code := p.approve(address, claims)
	parsed, _ := url.Parse(address)
	return callback(e, url.Values{"code": {code}, "state": {parsed.Query().Get("state")}}, state)
}

func callback(e *env, query url.Values, cookie *http.Cookie) (*url.URL, []*http.Cookie) {
	e.t.Helper()
	req, err := http.NewRequest("GET", e.url+"/api/auth/oidc/callback?"+query.Encode(), nil)
	require.NoError(e.t, err)
	if cookie != nil {
		req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	require.NoError(e.t, err)
	defer res.Body.Close()
	require.Equal(e.t, http.StatusFound, res.StatusCode)
	location, err := url.Parse(res.Header.Get("Location"))
	require.NoError(e.t, err)
	return location, res.Cookies()
}

func named(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name && c.Value != "" {
			return c
		}
	}
	return nil
}

func TestConnectedAccountsSignInWithTheProvider(t *testing.T) {
	p := newFakeProvider(t)
	e := newEnv(t, withOIDC(p, nil))
	e.req("GET", "/api/app/config/features").expect(200).has("sso", "Pocket ID").has("sso_only", false)
	user := e.register()
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("hasPassword", true)

	// Unknown to Ragbag (and nothing creates accounts): the app is told there is nothing connected
	person := map[string]any{"sub": "person-1", "email": "jo@example.com", "email_verified": true}
	location, _ := ssoRound(e, p, nil, person)
	require.Equal(t, "/login", location.Path)
	require.Equal(t, "signup", location.Query().Get("sso"))
	ticket, err := url.ParseQuery(location.Fragment)
	require.NoError(t, err)
	e.req("POST", "/api/auth/oidc/signup").json(fmt.Sprintf(`{"ticket":%q,"username":"jo"}`, ticket.Get("ticket"))).expect(403)

	// Connected in the settings, the same person signs in
	location, _ = ssoRound(e, p, user, person)
	require.Equal(t, "/settings/security?sso=connected", location.String())
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("sso", "jo@example.com")
	location, cookies := ssoRound(e, p, nil, person)
	require.Equal(t, "/login?sso=done", location.String())
	refresh := named(cookies, "refresh_token")
	require.NotNil(t, refresh)
	token := e.req("POST", "/api/auth/refresh").cookie(refresh).expect(200).str("access_token")
	e.req("GET", "/api/v1/users/me").bearer(token).expect(200).has("username", user.Username)

	// One account per person at the provider
	other := e.register()
	location, _ = ssoRound(e, p, other, person)
	require.Equal(t, "/settings/security", location.Path)
	require.Contains(t, location.Query().Get("sso_error"), "connected to another account")

	// Disconnected, the provider no longer signs them in
	e.req("DELETE", "/api/v1/users/me/oidc").as(user).expect(204)
	e.req("GET", "/api/v1/users/me").as(user).expect(200)
	require.False(t, e.req("GET", "/api/v1/users/me").as(user).expect(200).get("sso").Exists())
	location, _ = ssoRound(e, p, nil, person)
	require.Equal(t, "signup", location.Query().Get("sso"))
}

func TestTheCallbackOnlyFinishesSignInsOfThisBrowser(t *testing.T) {
	p := newFakeProvider(t)
	e := newEnv(t, withOIDC(p, nil))
	res := e.req("POST", "/api/auth/oidc/start").expect(200)
	address, _ := url.Parse(res.str("url"))
	code := p.approve(address.String(), map[string]any{"sub": "x", "email": "x@example.com"})
	state := address.Query().Get("state")

	// Without the cookie (a link somebody else started), with a forged state, or cancelled at the provider
	for _, attempt := range []struct {
		query  url.Values
		cookie *http.Cookie
	}{
		{url.Values{"code": {code}, "state": {state}}, nil},
		{url.Values{"code": {code}, "state": {"forged"}}, &http.Cookie{Name: "oidc_state", Value: "forged"}},
	} {
		location, cookies := callback(e, attempt.query, attempt.cookie)
		require.Equal(t, "/login", location.Path)
		require.NotEmpty(t, location.Query().Get("sso_error"))
		require.Nil(t, named(cookies, "refresh_token"))
	}
	// The flow is used up by the first try
	location, _ := callback(e, url.Values{"code": {code}, "state": {state}}, res.cookieNamed("oidc_state"))
	require.NotEmpty(t, location.Query().Get("sso_error"))

	res = e.req("POST", "/api/auth/oidc/start").expect(200)
	address, _ = url.Parse(res.str("url"))
	location, _ = callback(e, url.Values{"error": {"access_denied"}, "state": {address.Query().Get("state")}}, res.cookieNamed("oidc_state"))
	require.Contains(t, location.Query().Get("sso_error"), "cancelled")
}

func TestTheProviderCreatesAccountsWithAutoRegister(t *testing.T) {
	p := newFakeProvider(t)
	e := newEnv(t, withOIDC(p, func(o *config.OIDC) { o.AutoRegister = true }), func(cfg *config.Config) {
		cfg.RegistrationOpen = false
	})
	e.req("GET", "/api/app/config/features").expect(200).has("sso_signup", true)
	person := map[string]any{"sub": "person-2", "email": "Kim@Example.com", "email_verified": true,
		"preferred_username": "kim", "name": "Kim K."}
	location, _ := ssoRound(e, p, nil, person)
	fragment, err := url.ParseQuery(location.Fragment)
	require.NoError(t, err)
	require.Equal(t, "kim", fragment.Get("username"))
	require.Equal(t, "kim@example.com", fragment.Get("email"))
	ticket := fragment.Get("ticket")

	// The username is checked like at any registration, and the ticket stays usable for another try
	e.req("POST", "/api/auth/oidc/signup").json(fmt.Sprintf(`{"ticket":%q,"username":"has space"}`, ticket)).expect(400)
	res := e.req("POST", "/api/auth/oidc/signup").
		json(fmt.Sprintf(`{"ticket":%q,"username":"kim","displayName":"Kim K."}`, ticket)).expect(200)
	token := res.str("access_token")
	e.req("GET", "/api/v1/users/me").bearer(token).expect(200).
		has("username", "kim").has("email", "kim@example.com").has("hasPassword", false).has("sso", "kim@example.com")
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200).has("#", 1)
	e.req("POST", "/api/auth/oidc/signup").json(fmt.Sprintf(`{"ticket":%q,"username":"kim2"}`, ticket)).expect(400)

	// Without a password: confirmations ask for one first, setting the first one needs none
	kim := &session{Username: "kim", Token: token}
	e.req("POST", "/api/v1/users/me/delete").as(kim).json(`{"password":""}`).expect(400)
	e.req("DELETE", "/api/v1/users/me/oidc").as(kim).expect(409)
	e.req("POST", "/api/auth/password").as(kim).json(`{"currentPassword":"","newPassword":"secret123"}`).expect(200)
	e.login("kim", "secret123")

	// Back again: signed in, no second account
	location, _ = ssoRound(e, p, nil, person)
	require.Equal(t, "/login?sso=done", location.String())

	// Addresses of existing accounts are never taken over by somebody who can set them at the provider
	location, _ = ssoRound(e, p, nil, map[string]any{"sub": "intruder", "email": "kim@example.com", "email_verified": true})
	require.Contains(t, location.Query().Get("sso_error"), "account with kim@example.com already")
}

func TestOnlySingleSignOnLeavesPasswordsToAdmins(t *testing.T) {
	p := newFakeProvider(t)
	e := newEnv(t, withOIDC(p, func(o *config.OIDC) { o.Only, o.AutoRegister = true, true }))
	e.req("GET", "/api/app/config/features").expect(200).has("sso_only", true)
	e.req("POST", "/api/auth/register").json(`{"username":"new","email":"new@example.com","password":"secret123"}`).expect(403)

	location, _ := ssoRound(e, p, nil, map[string]any{"sub": "p", "email": "pat@example.com", "email_verified": true})
	fragment, _ := url.ParseQuery(location.Fragment)
	token := e.req("POST", "/api/auth/oidc/signup").json(fmt.Sprintf(`{"ticket":%q,"username":"pat"}`, fragment.Get("ticket"))).
		expect(200).str("access_token")
	pat := &session{Username: "pat", Token: token}
	// A password only confirms changes then, and the provider stays the way in
	pat.Token = e.req("POST", "/api/auth/password").as(pat).json(`{"currentPassword":"","newPassword":"secret123"}`).
		expect(200).str("access_token")
	e.req("POST", "/api/auth/login").json(`{"username":"pat","password":"secret123"}`).expect(403)
	e.req("DELETE", "/api/v1/users/me/oidc").bearer(e.login("admin", "test123").Token).expect(204)
	pat.Token = e.req("POST", "/api/auth/password").as(pat).json(`{"currentPassword":"secret123","newPassword":"secret456"}`).
		expect(200).str("access_token")
	e.req("DELETE", "/api/v1/users/me/oidc").as(pat).expect(409)

	// Instance admins keep their password, in case the provider is down
	e.login("admin", "test123")
}

func TestStartingNeedsAReachableProvider(t *testing.T) {
	p := newFakeProvider(t)
	e := newEnv(t, withOIDC(p, nil))
	p.server.Close()
	res := e.req("POST", "/api/auth/oidc/start").expect(502)
	require.True(t, strings.Contains(res.Body, "cannot be reached"), res.Body)

	// Without OIDC_ISSUER there is nothing to start
	plain := newEnv(t)
	plain.req("POST", "/api/auth/oidc/start").expect(404)
	require.False(t, plain.req("GET", "/api/app/config/features").expect(200).get("sso").Exists())
}

func TestAnInviteLetsTheProviderCreateTheAccount(t *testing.T) {
	p := newFakeProvider(t)
	e := newEnv(t, withOIDC(p, nil), func(cfg *config.Config) { cfg.RegistrationOpen = false })
	admin := e.login("admin", "test123")
	space := e.firstSpace(admin)
	invite := e.invite(admin, space, "CONTRIBUTOR")

	location, _ := ssoRound(e, p, nil, map[string]any{"sub": "guest", "email": "guest@example.com", "email_verified": true})
	fragment, _ := url.ParseQuery(location.Fragment)
	token := e.req("POST", "/api/auth/oidc/signup").
		json(fmt.Sprintf(`{"ticket":%q,"username":"guest","inviteToken":%q}`, fragment.Get("ticket"), invite)).
		expect(200).str("access_token")
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200).has(fmt.Sprintf(`#(id==%q).role`, space), "CONTRIBUTOR")
}
