package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityHeaders(t *testing.T) {
	e := newEnv(t)
	page := e.req("GET", "/").expect(200)
	csp := page.Header.Get("Content-Security-Policy")
	require.Contains(t, csp, "script-src 'self'")
	require.Contains(t, csp, "frame-ancestors 'none'")
	// Link previews and favicons come from the linked sites, also by plain HTTP
	require.Contains(t, csp, "img-src 'self' data: blob: https: http:")
	require.Equal(t, "DENY", page.Header.Get("X-Frame-Options"))
	require.Equal(t, "nosniff", page.Header.Get("X-Content-Type-Options"))
	require.Equal(t, "no-referrer", page.Header.Get("Referrer-Policy"))
	require.Equal(t, "same-origin", page.Header.Get("Cross-Origin-Opener-Policy"))
	require.NotEmpty(t, page.Header.Get("Permissions-Policy"))
	// Plain HTTP (a homelab): no HSTS
	require.Empty(t, page.Header.Get("Strict-Transport-Security"))

	// The bookmarklet's window keeps its opener to close itself
	require.Empty(t, e.req("GET", "/add").expect(200).Header.Get("Cross-Origin-Opener-Policy"))

	// API answers are never pages and never cached, errors included
	admin := e.login("admin", "test123")
	for _, res := range []*response{e.req("GET", "/api/v1/users/me").as(admin).expect(200),
		e.req("GET", "/api/v1/users/me").expect(401)} {
		require.Equal(t, "default-src 'none'; frame-ancestors 'none'", res.Header.Get("Content-Security-Policy"))
		require.Equal(t, "no-store", res.Header.Get("Cache-Control"))
	}

	// Behind a proxy that speaks HTTPS
	secure := e.req("GET", "/").header("X-Forwarded-Proto", "https").expect(200)
	require.Equal(t, "max-age=31536000", secure.Header.Get("Strict-Transport-Security"))
}

func TestImagesKeepTheirOwnHeaders(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	avatar := e.req("PUT", "/api/v1/users/me/avatar").as(user).
		multipart(map[string]any{"file": pngImage(t, 64, 64)}).expect(200).str("avatarId")
	res := e.req("GET", "/api/v1/images/%s", avatar).expect(200)
	require.True(t, strings.HasPrefix(res.Header.Get("Content-Security-Policy"), "default-src 'none'; sandbox"))
	require.Contains(t, res.Header.Get("Cache-Control"), "immutable")
}
