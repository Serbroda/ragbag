package server_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

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
	e.req("GET", "/api/app/config/features").expect(200).has("legal_imprint", "/app/legal/imprint")
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
