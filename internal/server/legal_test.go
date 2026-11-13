package server_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (e *env) writeLegal(page, content string) {
	e.t.Helper()
	dir := filepath.Join(e.cfg.DataDir, "legal")
	require.NoError(e.t, os.MkdirAll(dir, 0o755))
	require.NoError(e.t, os.WriteFile(filepath.Join(dir, page+".md"), []byte(content), 0o644))
}

func registerBody(terms string) string {
	username := "u" + uuid.NewString()[:8]
	return fmt.Sprintf(`{"username":%q,"email":"%s@example.com","password":"secret123"%s}`, username, username, terms)
}

func TestLegalPages(t *testing.T) {
	e := newEnv(t)
	// None by default: nothing linked, nothing to accept
	flags := e.req("GET", "/api/app/config/features").expect(200)
	require.False(t, flags.get("legal_imprint").Exists())
	e.req("GET", "/api/app/legal/imprint").expect(404)
	e.req("POST", "/api/auth/register").json(registerBody("")).expect(200)

	e.writeLegal("imprint", "# Impressum\n\nMax Mustermann, **Musterstraße 1**\n\n<script>alert(1)</script>\n\n| a | b |\n|---|---|\n| 1 | 2 |\n")
	e.req("GET", "/api/app/config/features").expect(200).has("legal_imprint", "/legal/imprint")
	page := e.req("GET", "/api/app/legal/imprint").expect(200)
	page.has("title", "Impressum").has("page", "imprint")
	html := page.str("html")
	require.Contains(t, html, "<strong>Musterstraße 1</strong>")
	require.Contains(t, html, "<table>")
	require.NotContains(t, html, "<script>")
	require.NotContains(t, html, "Impressum", "the heading became the title")
	e.req("GET", "/api/app/legal/secrets").expect(404)
	require.NotEqual(t, 200, e.req("GET", "/api/app/legal/%s", "..%2F..%2Fragbag.db").send().Status)
}

func TestTermsOfUseAreAcceptedAtRegistration(t *testing.T) {
	e := newEnv(t)
	e.writeLegal("terms", "# Terms\n\nBe nice.\n")
	e.req("POST", "/api/auth/register").json(registerBody("")).expect(400)
	e.req("POST", "/api/auth/register").json(registerBody(`,"acceptTerms":false`)).expect(400)
	id := e.req("POST", "/api/auth/register").json(registerBody(`,"acceptTerms":true`)).expect(200).str("id")
	var accepted int
	require.NoError(t, e.db.Get(&accepted, e.db.Rebind(`SELECT COUNT(*) FROM users WHERE id = ? AND terms_accepted_at IS NOT NULL`), id))
	require.Equal(t, 1, accepted)
}

func TestLegalPagesElsewhere(t *testing.T) {
	e := newEnv(t, func(cfg *config.Config) {
		cfg.LegalURLs = map[string]string{"privacy": "https://example.com/datenschutz", "terms": "https://example.com/agb"}
	})
	e.req("GET", "/api/app/config/features").expect(200).
		has("legal_privacy", "https://example.com/datenschutz").has("legal_terms", "https://example.com/agb")
	e.req("POST", "/api/auth/register").json(registerBody("")).expect(400)
	admin := e.login("admin", "test123")
	e.req("GET", "/api/v1/admin/overview").as(admin).expect(200).
		has("settings.legalTerms", "https://example.com/agb").has("settings.legalImprint", "")
}

func TestChangedTermsHaveToBeAcceptedAgain(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "test123")
	user := e.register()
	space := e.firstSpace(user)
	token, _ := e.apiToken(user, "READ")

	// Without terms there is nothing to change
	e.req("POST", "/api/v1/admin/terms").as(admin).json(`{}`).expect(400)
	e.writeLegal("terms", "# Terms\n\nBe nice.\n")
	e.req("POST", "/api/v1/admin/terms").as(user).json(`{}`).expect(403)

	// Announced: everything works, the app shows when
	next := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	state := e.req("POST", "/api/v1/admin/terms").as(admin).json(fmt.Sprintf(`{"effectiveAt":%q}`, next.Format(time.RFC3339))).
		expect(200).has("accounts", 2)
	require.True(t, next.Equal(state.get("next").Time()))
	me := e.req("GET", "/api/v1/users/me").as(user).expect(200)
	require.False(t, me.get("termsPending").Exists())
	require.True(t, next.Equal(me.get("termsChangeAt").Time()))
	e.req("GET", "/api/v1/spaces").as(user).expect(200)
	e.req("DELETE", "/api/v1/admin/terms/next").as(admin).expect(200)
	e.req("DELETE", "/api/v1/admin/terms/next").as(admin).expect(404)

	// In effect: neither the app nor API tokens work until the user accepts them
	state = e.req("POST", "/api/v1/admin/terms").as(admin).json(`{}`).expect(200)
	require.True(t, state.get("inForceSince").Exists())
	require.Equal(t, int64(0), state.get("accepted").Int())
	denied := e.req("GET", "/api/v1/spaces").as(user).expect(403)
	require.Equal(t, "accept", denied.Header.Get("X-Ragbag-Terms"))
	denied = e.req("GET", "/api/v1/spaces/%s/collections", space).bearer(token).expect(403)
	require.Equal(t, "accept", denied.Header.Get("X-Ragbag-Terms"))
	e.req("POST", "/api/v1/users/me/terms").bearer(token).expect(403)
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("termsPending", true)

	e.req("POST", "/api/v1/users/me/terms").as(user).expect(204)
	e.req("GET", "/api/v1/spaces").as(user).expect(200)
	e.req("GET", "/api/v1/spaces/%s/collections", space).bearer(token).expect(200)
	require.False(t, e.req("GET", "/api/v1/users/me").as(user).expect(200).get("termsPending").Exists())

	// New accounts accept them when registering; who does not agree may still delete the account
	e.req("POST", "/api/auth/register").json(registerBody(`,"acceptTerms":true`)).expect(200)
	e.req("POST", "/api/v1/users/me/terms").as(admin).expect(204)
	e.req("GET", "/api/v1/admin/terms").as(admin).expect(200).has("accepted", 3).has("accounts", 3)
	e.req("POST", "/api/v1/admin/terms").as(admin).json(`{}`).expect(200)
	e.req("POST", "/api/v1/users/me/delete").as(user).json(`{"password":"secret123"}`).expect(204)
}
