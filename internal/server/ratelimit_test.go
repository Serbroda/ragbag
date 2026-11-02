package server_test

import (
	"fmt"
	"testing"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

func withRateLimits(cfg *config.Config) { cfg.RateLimits = true }

// from sends the request as if it came through a proxy from that address (the test server is loopback, which
// counts as an own proxy).
func (r *request) from(ip string) *request { return r.header("X-Forwarded-For", ip) }

func loginBody(username, password string) string {
	return fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
}

func TestLoginsArePacedPerAddress(t *testing.T) {
	e := newEnv(t, withRateLimits)
	for i := range 10 {
		e.req("POST", "/api/auth/login").from("203.0.113.7").json(loginBody(fmt.Sprintf("nobody%d", i), "wrong-password")).expect(401)
	}
	res := e.req("POST", "/api/auth/login").from("203.0.113.7").json(loginBody("nobody", "wrong-password")).expect(429)
	require.NotEmpty(t, res.Header.Get("Retry-After"))
	require.NotEmpty(t, res.str("detail"))
	// Another address is not affected, and IPv6 counts by its /64
	e.req("POST", "/api/auth/login").from("203.0.113.8").json(loginBody("somebody", "wrong-password")).expect(401)
	for i := range 10 {
		e.req("POST", "/api/auth/login").from(fmt.Sprintf("2001:db8:1:2::%x", i+1)).json(loginBody(fmt.Sprintf("v6user%d", i), "wrong-password")).expect(401)
	}
	e.req("POST", "/api/auth/login").from("2001:db8:1:2::ffff").json(loginBody("v6user", "wrong-password")).expect(429)
	e.req("POST", "/api/auth/login").from("2001:db8:1:3::1").json(loginBody("v6user", "wrong-password")).expect(401)
}

func TestFailedLoginsBlockTheAccountFromEverywhere(t *testing.T) {
	e := newEnv(t, withRateLimits)
	user := e.register()
	for i := range 10 {
		// Username and email are the same account
		login := user.Username
		if i%2 == 1 {
			login = user.Username + "@example.com"
		}
		e.req("POST", "/api/auth/login").from(fmt.Sprintf("198.51.100.%d", i+1)).json(loginBody(login, "wrong-password")).expect(401)
	}
	// Even the right password waits now, from a new address
	e.req("POST", "/api/auth/login").from("198.51.100.200").json(loginBody(user.Username, testPassword)).expect(429)
	// The session that is already signed in keeps working
	e.req("GET", "/api/v1/users/me").as(user).expect(200)
	// Other accounts can still sign in
	other := e.register()
	e.req("POST", "/api/auth/login").from("198.51.100.200").json(loginBody(other.Username, testPassword)).expect(200)
}

func TestRegistrationsArePacedPerAddress(t *testing.T) {
	e := newEnv(t, withRateLimits)
	for i := range 6 {
		status := 200
		if i == 5 {
			status = 429
		}
		e.req("POST", "/api/auth/register").from("192.0.2.50").
			json(fmt.Sprintf(`{"username":"bot%d","email":"bot%d@example.com","password":"secret123"}`, i, i)).expect(status)
	}
}

func TestLoadingPagesIsPacedPerUser(t *testing.T) {
	e := newEnv(t, withRateLimits)
	user := e.register()
	collection := e.createCollection(user, e.firstSpace(user), "Links", "")
	for range 60 {
		e.req("POST", "/api/v1/collections/%s/bookmarks/metadata", collection).as(user).json(`{"url":"127.0.0.1/page"}`).expect(200)
	}
	e.req("POST", "/api/v1/collections/%s/bookmarks/metadata", collection).as(user).json(`{"url":"127.0.0.1/page"}`).expect(429)
	// Links are still saved, only without loading their details
	e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).json(`{"url":"127.0.0.1/saved"}`).
		expect(200).has("title", "https://127.0.0.1/saved")
	// Other users have their own budget
	other := e.register()
	own := e.createCollection(other, e.firstSpace(other), "Links", "")
	e.req("POST", "/api/v1/collections/%s/bookmarks/metadata", own).as(other).json(`{"url":"127.0.0.1/page"}`).expect(200)
}

func TestRateLimitsCanBeTurnedOff(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	for range 15 {
		e.req("POST", "/api/auth/login").from("203.0.113.7").json(loginBody(user.Username, "wrong-password")).expect(401)
	}
	e.login(user.Username, testPassword)
}
