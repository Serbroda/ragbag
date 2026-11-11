package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/stretchr/testify/require"
)

// codeAt is the code of the app, offset by whole 30 second windows (each code works only once).
func codeAt(t *testing.T, secret string, windows int) string {
	code, err := auth.TOTPCode(secret, time.Now().Add(time.Duration(windows)*30*time.Second))
	require.NoError(t, err)
	return code
}

func (e *env) loginWith(s *session, code string) *request {
	body := fmt.Sprintf(`{"username":%q,"password":%q}`, s.Username, testPassword)
	if code != "" {
		body = fmt.Sprintf(`{"username":%q,"password":%q,"code":%q}`, s.Username, testPassword, code)
	}
	return e.req("POST", "/api/auth/login").json(body)
}

// enableTwoFactor sets it up like the profile dialog and returns the secret and the recovery codes.
func (e *env) enableTwoFactor(s *session) (string, []string) {
	secret := e.req("POST", "/api/v1/users/me/two-factor/setup").as(s).
		json(fmt.Sprintf(`{"currentPassword":%q}`, testPassword)).expect(200).str("secret")
	res := e.req("POST", "/api/v1/users/me/two-factor/enable").as(s).
		json(fmt.Sprintf(`{"code":%q}`, codeAt(e.t, secret, 0))).expect(200)
	var codes []string
	for _, code := range res.get("recoveryCodes").Array() {
		codes = append(codes, code.String())
	}
	require.Len(e.t, codes, 10)
	return secret, codes
}

func TestTwoFactorIsOffUntilTheFirstCodeConfirmsIt(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("twoFactor", false)

	e.req("POST", "/api/v1/users/me/two-factor/setup").as(user).json(`{"currentPassword":"wrong"}`).expect(400)
	e.req("POST", "/api/v1/users/me/two-factor/enable").as(user).json(`{"code":"123456"}`).expect(400)
	setup := e.req("POST", "/api/v1/users/me/two-factor/setup").as(user).
		json(fmt.Sprintf(`{"currentPassword":%q}`, testPassword)).expect(200)
	secret := setup.str("secret")
	require.Contains(t, setup.str("uri"), "otpauth://totp/Ragbag:"+user.Username+"?")
	require.Contains(t, setup.str("uri"), "secret="+secret)

	// Started but not confirmed: the login does not ask for anything, a wrong code turns nothing on
	e.loginWith(user, "").expect(200)
	e.req("POST", "/api/v1/users/me/two-factor/enable").as(user).json(`{"code":"000000"}`).expect(400)
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("twoFactor", false)

	e.req("POST", "/api/v1/users/me/two-factor/enable").as(user).
		json(fmt.Sprintf(`{"code":%q}`, codeAt(t, secret, 0))).expect(200).has("recoveryCodes.#", 10)
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("twoFactor", true)
	e.req("POST", "/api/v1/users/me/two-factor/setup").as(user).
		json(fmt.Sprintf(`{"currentPassword":%q}`, testPassword)).expect(409)
	// The session that turned it on stays signed in
	e.req("GET", "/api/v1/spaces").as(user).expect(200)
}

func TestLoginNeedsTheCodeOnceTwoFactorIsOn(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	secret, _ := e.enableTwoFactor(user)

	missing := e.loginWith(user, "").expect(401)
	require.Equal(t, "required", missing.Header.Get("X-Ragbag-Two-Factor"))
	require.Nil(t, missing.cookieNamed("refresh_token"))
	wrong := e.loginWith(user, "000000").expect(401)
	require.Equal(t, "required", wrong.Header.Get("X-Ragbag-Two-Factor"))
	// A wrong password never tells whether the account has a second factor
	bad := e.req("POST", "/api/auth/login").json(fmt.Sprintf(`{"username":%q,"password":"nope","code":"000000"}`, user.Username)).expect(401)
	require.Empty(t, bad.Header.Get("X-Ragbag-Two-Factor"))

	// A code signs in once (the one that confirmed the setup is used up too, so take the next)
	next := codeAt(t, secret, 1)
	signedIn := e.loginWith(user, next).expect(200)
	require.NotEmpty(t, signedIn.str("access_token"))
	e.req("GET", "/api/v1/spaces").bearer(signedIn.str("access_token")).expect(200)
	e.loginWith(user, next).expect(401)
}

func TestRecoveryCodesWorkOnceAndResetLinksDoNotGetAroundTheCode(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	_, recovery := e.enableTwoFactor(user)

	e.loginWith(user, "  "+recovery[0][:5]+" "+recovery[0][6:]+" ").expect(200)
	e.loginWith(user, recovery[0]).expect(401)
	e.loginWith(user, recovery[1]).expect(200)

	// An admin's reset link sets a new password, which still does not sign in alone
	admin := e.login("admin", "test123")
	link := e.req("POST", "/api/v1/admin/users/%s/password-reset", user.UserID).as(admin).expect(200).str("token")
	e.req("POST", "/api/auth/password-reset/%s", link).json(`{"newPassword":"another-secret"}`).expect(204)
	login := func(code string) *request {
		return e.req("POST", "/api/auth/login").
			json(fmt.Sprintf(`{"username":%q,"password":"another-secret","code":%q}`, user.Username, code))
	}
	login("").expect(401)
	login(recovery[2]).expect(200)
}

func TestTooManyWrongCodesPauseTheSecondFactor(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	secret, _ := e.enableTwoFactor(user)
	for range 4 {
		e.loginWith(user, "000000").expect(401)
	}
	e.loginWith(user, "000000").expect(429)
	// Even the right code has to wait now
	e.loginWith(user, codeAt(t, secret, 1)).expect(429)
}

func TestTurningTwoFactorOffNeedsPasswordAndCode(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	secret, recovery := e.enableTwoFactor(user)
	disable := func(password, code string) *request {
		return e.req("POST", "/api/v1/users/me/two-factor/disable").as(user).
			json(fmt.Sprintf(`{"currentPassword":%q,"code":%q}`, password, code))
	}
	disable("wrong", codeAt(t, secret, 1)).expect(400)
	disable(testPassword, "000000").expect(400)
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("twoFactor", true)
	disable(testPassword, recovery[0]).expect(204)
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("twoFactor", false)
	e.loginWith(user, "").expect(200)

	// Turning it on again gives a new secret and new recovery codes
	again, _ := e.enableTwoFactor(user)
	require.NotEqual(t, secret, again)
	e.loginWith(user, recovery[1]).expect(401)
}

func TestAdminsResetTheSecondFactorOfOthers(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	e.enableTwoFactor(user)
	other := e.register()
	e.req("DELETE", "/api/v1/admin/users/%s/two-factor", user.UserID).as(other).expect(403)

	admin := e.login("admin", "test123")
	e.req("GET", "/api/v1/admin/users").as(admin).expect(200).
		has(fmt.Sprintf(`#(id==%q).twoFactor`, user.UserID), true).has(fmt.Sprintf(`#(id==%q).twoFactor`, other.UserID), false)
	e.req("DELETE", "/api/v1/admin/users/%s/two-factor", user.UserID).as(admin).expect(204)
	e.loginWith(user, "").expect(200)
	e.req("GET", "/api/v1/admin/users").as(admin).expect(200).has(fmt.Sprintf(`#(id==%q).twoFactor`, user.UserID), false)
}
