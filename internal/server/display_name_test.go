package server_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisplayNamesAreSetAtRegistrationOrLater(t *testing.T) {
	e := newEnv(t)
	username := newUsername()
	e.req("POST", "/api/auth/register").
		json(fmt.Sprintf(`{"username":%q,"email":"%s@example.com","password":%q,"displayName":"  Max Power 🎮 "}`,
			username, username, testPassword)).
		expect(200).has("displayName", "Max Power 🎮").has("username", username)
	// Signing in stays with the username; the first space is named after the display name
	max := e.login(username, testPassword)
	e.req("GET", "/api/v1/spaces").as(max).expect(200).has("0.name", "Max Power 🎮's Space")

	setName := func(body string) *request {
		return e.req("PUT", "/api/v1/users/me/display-name").as(max).json(body)
	}
	setName(`{"displayName":"Maximilian"}`).expect(200).has("displayName", "Maximilian")
	e.req("GET", "/api/v1/users/me").as(max).expect(200).has("displayName", "Maximilian").has("username", username)
	setName(fmt.Sprintf(`{"displayName":%q}`, strings.Repeat("ü", 51))).expect(400)
	setName(`{"displayName":"Max\nPower"}`).expect(400)
	setName(fmt.Sprintf(`{"displayName":%q}`, strings.Repeat("ü", 50))).expect(200)

	// Empty removes it
	res := setName(`{"displayName":"   "}`).expect(200)
	require.False(t, res.get("displayName").Exists(), res.Body)
	res = setName(`{}`).expect(200)
	require.False(t, res.get("displayName").Exists(), res.Body)

	e.req("POST", "/api/auth/register").
		json(fmt.Sprintf(`{"username":%q,"email":"x%s@example.com","password":%q,"displayName":%q}`,
			newUsername(), username, testPassword, strings.Repeat("x", 51))).expect(400)
}

// Display names are not unique, so lists of people carry both; everything else shows the display name.
func TestOthersSeeTheDisplayName(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	e.req("PUT", "/api/v1/users/me/display-name").as(owner).json(`{"displayName":"Danny R."}`).expect(200)
	space := e.firstSpace(owner)
	e.join(member, space, "CONTRIBUTOR", owner)
	collection := e.createCollection(owner, space, "Links", "")
	bookmark := e.createBookmark(owner, collection)

	e.req("GET", "/api/v1/spaces/%s/members", space).as(member).expect(200).
		has(fmt.Sprintf("#(userId==%q).displayName", owner.UserID), "Danny R.").
		has(fmt.Sprintf("#(userId==%q).username", owner.UserID), owner.Username)
	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(member).expect(200).has("0.createdBy", "Danny R.")
	e.req("GET", "/api/v1/spaces/%s/news", space).as(member).expect(200).has("0.addedBy", `["Danny R."]`)
	e.req("PUT", "/api/v1/bookmarks/%s/reactions", bookmark).as(owner).json(`{"emoji":"❤️","reacted":true}`).
		expect(200).has("0.users", `["Danny R."]`)

	token := e.invite(owner, space, "VIEWER")
	e.req("GET", "/api/v1/invites/%s", token).as(member).expect(200).has("invitedBy", "Danny R.")
	e.req("GET", "/api/auth/invites/%s", token).expect(200).has("invitedBy", "Danny R.")
	require.Contains(t, e.req("GET", "/app/invite/%s", token).expect(200).Body, "Danny R. invited you to")

	// Admins see both
	admin := e.login("admin", "test123")
	e.req("GET", "/api/v1/admin/users").as(admin).expect(200).
		has(fmt.Sprintf("#(id==%q).displayName", owner.UserID), "Danny R.").
		has(fmt.Sprintf("#(id==%q).username", owner.UserID), owner.Username)
}

func TestDeletingAnAccountRemovesItsDisplayName(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(member, space, "CONTRIBUTOR", owner)
	collection := e.createCollection(owner, space, "Links", "")
	e.req("PUT", "/api/v1/users/me/display-name").as(member).json(`{"displayName":"Erika Mustermann"}`).expect(200)
	e.createBookmark(member, collection)

	e.req("POST", "/api/v1/users/me/delete").as(member).json(fmt.Sprintf(`{"password":%q}`, testPassword)).expect(204)
	createdBy := e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(owner).expect(200).str("0.createdBy")
	require.True(t, strings.HasPrefix(createdBy, "deleted-"), createdBy)
}
