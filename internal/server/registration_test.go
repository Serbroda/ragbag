package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func closedRegistration(cfg *config.Config) { cfg.RegistrationOpen = false }

// inviteRegistration is a registration with an invite; the username is also the start of the email address.
func inviteRegistration(username, inviteToken string) string {
	return fmt.Sprintf(`{"username":%q,"email":"%s@example.com","password":%q,"inviteToken":%q}`,
		username, username, testPassword, inviteToken)
}

func newUsername() string { return "u" + uuid.NewString()[:8] }

// spaceRoles maps the names of the user's spaces to their role there.
func (e *env) spaceRoles(s *session) map[string]string {
	e.t.Helper()
	roles := map[string]string{}
	for _, space := range e.req("GET", "/api/v1/spaces").as(s).expect(200).get("@this").Array() {
		roles[space.Get("name").String()] = space.Get("role").String()
	}
	return roles
}

func TestInvitesOfAdminsRegisterWhileRegistrationIsClosed(t *testing.T) {
	e := newEnv(t, closedRegistration)
	admin := e.login("admin", "test123")
	adminSpace := e.firstSpace(admin)
	spaceName := e.req("GET", "/api/v1/spaces/%s", adminSpace).as(admin).expect(200).str("name")

	e.req("POST", "/api/auth/register").json(inviteRegistration(newUsername(), "")).expect(403)
	e.req("GET", "/api/auth/invites/unknown").expect(404)
	e.req("POST", "/api/auth/register").json(inviteRegistration(newUsername(), "unknown")).expect(404)

	// Anybody may invite to their own spaces, but only admins let new people into the instance
	token := e.req("POST", "/api/v1/invites").as(admin).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"role":"CONTRIBUTOR","maxUses":1}`, adminSpace)).
		expect(201).str("token")
	e.req("GET", "/api/auth/invites/%s", token).expect(200).
		has("registration", true).has("valid", true).has("invitedBy", "admin").has("targetName", spaceName)

	username := newUsername()
	e.req("POST", "/api/auth/register").json(inviteRegistration(username, token)).expect(200)
	newcomer := e.login(username, testPassword)
	roles := e.spaceRoles(newcomer)
	require.Equal(t, "CONTRIBUTOR", roles[spaceName])
	require.Len(t, roles, 2, "own first space as well")

	// Used up
	e.req("GET", "/api/auth/invites/%s", token).expect(200).has("registration", false).has("valid", false)
	e.req("POST", "/api/auth/register").json(inviteRegistration(newUsername(), token)).expect(409)
}

func TestInvitesOfOtherUsersOnlyRegisterWhileRegistrationIsOpen(t *testing.T) {
	open := newEnv(t)
	member := open.register()
	token := open.invite(member, open.firstSpace(member), "VIEWER")
	open.req("GET", "/api/auth/invites/%s", token).expect(200).has("registration", true)
	username := newUsername()
	open.req("POST", "/api/auth/register").json(inviteRegistration(username, token)).expect(200)
	memberSpace := open.req("GET", "/api/v1/spaces/%s", open.firstSpace(member)).as(member).expect(200).str("name")
	require.Equal(t, "VIEWER", open.spaceRoles(open.login(username, testPassword))[memberSpace])

	closed := newEnv(t, closedRegistration)
	admin := closed.login("admin", "test123")
	// Accounts of a closed instance come from admin invites
	username = newUsername()
	closed.req("POST", "/api/auth/register").json(inviteRegistration(username, closed.invite(admin, closed.firstSpace(admin), "VIEWER"))).expect(200)
	member = closed.login(username, testPassword)
	token = closed.invite(member, closed.firstSpace(member), "VIEWER")
	closed.req("GET", "/api/auth/invites/%s", token).expect(200).has("registration", false).has("valid", true)
	closed.req("POST", "/api/auth/register").json(inviteRegistration(newUsername(), token)).expect(403)
	// Nothing was used up
	closed.req("GET", "/api/v1/spaces/%s/invites", closed.firstSpace(member)).as(member).expect(200).has("0.usedCount", 0)
}

func TestInvitedAccountsJoinTheSpaceWithTheirConfirmation(t *testing.T) {
	e, box := newMailEnv(t, closedRegistration)
	admin := e.login("admin", "test123")
	adminSpace := e.firstSpace(admin)
	token := e.invite(admin, adminSpace, "VIEWER")

	username := newUsername()
	email := username + "@example.com"
	e.req("POST", "/api/auth/register").json(inviteRegistration(username, token)).expect(200)
	_, first := box.await(t, 1, email)
	members := e.req("GET", "/api/v1/spaces/%s/members", adminSpace).as(admin).expect(200).get("#").Int()

	// A new link (sent again before confirming) still joins the space
	e.sentVerificationAgo(3 * time.Minute)
	e.req("POST", "/api/auth/resend-verification").json(loginBody(username, testPassword)).expect(204)
	_, second := box.await(t, 2, email)
	require.NotEqual(t, first, second)
	access := e.req("POST", "/api/auth/verify-email/%s", second).expect(200).str("access_token")
	e.req("GET", "/api/v1/spaces/%s/members", adminSpace).as(admin).expect(200).has("#", members+1)
	e.req("GET", "/api/v1/spaces/%s", adminSpace).bearer(access).expect(200)

	// Revoked before the confirmation: only the own space
	username = newUsername()
	revoked := e.invite(admin, adminSpace, "VIEWER")
	e.req("POST", "/api/auth/register").json(inviteRegistration(username, revoked)).expect(200)
	_, link := box.await(t, 3, username+"@example.com")
	e.req("DELETE", "/api/v1/invites/%s", revoked).as(admin).expect(204)
	access = e.req("POST", "/api/auth/verify-email/%s", link).expect(200).str("access_token")
	e.req("GET", "/api/v1/spaces").bearer(access).expect(200).has("#", 1)
}
