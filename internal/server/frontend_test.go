package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheAppIsServedAtTheRoot(t *testing.T) {
	e := newEnv(t)
	for _, path := range []string{"/", "/login", "/space/some-id/collections/other-id"} {
		require.Contains(t, e.req("GET", "%s", path).expect(200).Body, "<title>", path)
	}
	// Files that do not exist are not the app, and the API stays the API
	e.req("GET", "/missing.png").expect(404)
	admin := e.login("admin", "test123")
	require.NotContains(t, e.req("GET", "/api/v1/unknown").as(admin).expect(404).Body, "<title>")
	for _, path := range []string{"/api", "/api/v1/unknown"} {
		res := e.req("GET", "%s", path).send()
		require.NotEqual(t, http.StatusOK, res.Status, path)
		require.NotContains(t, res.Body, "<title>", path)
	}
}

func TestAddressesOfTheOldAppPrefixMoveForGood(t *testing.T) {
	e := newEnv(t)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for from, to := range map[string]string{
		"/app":                    "/",
		"/app/":                   "/",
		"/app/share/abc":          "/share/abc",
		"/app/add?url=x&title=y":  "/add?url=x&title=y",
		"/app//evil.example/path": "/evil.example/path",
	} {
		res, err := client.Get(e.url + from)
		require.NoError(t, err)
		res.Body.Close()
		require.Equal(t, http.StatusMovedPermanently, res.StatusCode, from)
		require.Equal(t, to, res.Header.Get("Location"), from)
	}
}
