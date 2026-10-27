package server_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func (e *env) changePassword(s *session, current, next string) *request {
	return e.req("POST", "/api/auth/password").as(s).json(fmt.Sprintf(`{"currentPassword":%q,"newPassword":%q}`, current, next))
}

func TestChangePasswordSignsOutOtherSessionsAndKeepsThisOne(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	res := e.changePassword(user, "secret123", "a-new-password").expect(200)
	require.NotNil(t, res.cookieNamed("refresh_token"))

	e.req("GET", "/api/v1/users/me").bearer(res.str("access_token")).expect(200)
	// Tokens issued before the change (other devices) are invalid
	e.req("GET", "/api/v1/users/me").as(user).expect(401)
	e.req("POST", "/api/auth/refresh").cookie(user.Refresh).expect(401)

	e.req("POST", "/api/auth/login").json(fmt.Sprintf(`{"username":%q,"password":"secret123"}`, user.Username)).expect(401)
	e.login(user.Username, "a-new-password")
}

func TestChangePasswordRequiresTheCurrentPasswordAndAValidNewOne(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	// 400, not 401: the client would treat a 401 as an expired session
	e.changePassword(user, "wrong-password", "a-new-password").expect(400)
	e.changePassword(user, "secret123", "short").expect(400)
	e.changePassword(user, "secret123", strings.Repeat("x", 73)).expect(400)
	e.req("GET", "/api/v1/users/me").as(user).expect(200)
}

func (e *env) createReset(admin *session, userID string) *request {
	return e.req("POST", "/api/v1/admin/users/%s/password-reset", userID).as(admin)
}

func (e *env) resetPassword(token, password string) *request {
	return e.req("POST", "/api/auth/password-reset/%s", token).json(fmt.Sprintf(`{"newPassword":%q}`, password))
}

func TestResetLinkSetsANewPasswordOnceAndSignsOutEverywhere(t *testing.T) {
	e := newEnv(t)
	admin, user := e.login("admin", "test123"), e.register()
	res := e.createReset(admin, user.UserID).expect(200)
	require.NotEmpty(t, res.str("expiresAt"))
	token := res.str("token")

	e.req("GET", "/api/auth/password-reset/%s", token).expect(200).has("valid", true).has("username", user.Username)
	e.resetPassword(token, "short").expect(400)
	e.resetPassword(token, "reset-password-1").expect(204)

	e.login(user.Username, "reset-password-1")
	e.req("GET", "/api/v1/users/me").as(user).expect(401)
	// Used up
	e.req("GET", "/api/auth/password-reset/%s", token).expect(200).has("valid", false)
	e.resetPassword(token, "reset-password-2").expect(400)
}

func TestANewResetLinkReplacesTheOldOne(t *testing.T) {
	e := newEnv(t)
	admin, user := e.login("admin", "test123"), e.register()
	first := e.createReset(admin, user.UserID).expect(200).str("token")
	second := e.createReset(admin, user.UserID).expect(200).str("token")
	e.resetPassword(first, "reset-password-1").expect(400)
	e.resetPassword(second, "reset-password-1").expect(204)
}

func TestOnlyInstanceAdminsManageUsers(t *testing.T) {
	e := newEnv(t)
	admin, user, other := e.login("admin", "test123"), e.register(), e.register()

	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("instanceAdmin", false)
	e.req("GET", "/api/v1/admin/users").as(user).expect(403)
	e.createReset(user, other.UserID).expect(403)

	e.req("GET", "/api/v1/users/me").as(admin).expect(200).has("instanceAdmin", true)
	e.req("GET", "/api/v1/admin/users").as(admin).expect(200).
		has(fmt.Sprintf(`#(username==%q).spaces`, user.Username), 1).has(`#(username=="admin").instanceAdmin`, true)
	e.createReset(admin, "00000000-0000-0000-0000-000000000000").expect(404)
}

func TestUnknownResetLinksAreInvalid(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/auth/password-reset/not-a-token").expect(200).has("valid", false)
	e.resetPassword("not-a-token", "reset-password-1").expect(400)
}
