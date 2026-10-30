package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type spaceFixture struct {
	*env
	owner, admin, contributor, viewer *session
	space                             string
}

func newSpaceFixture(t *testing.T) *spaceFixture {
	e := newEnv(t)
	f := &spaceFixture{env: e, owner: e.register(), admin: e.register(), contributor: e.register(), viewer: e.register()}
	f.space = e.firstSpace(f.owner)
	e.join(f.admin, f.space, "ADMIN", f.owner)
	e.join(f.contributor, f.space, "CONTRIBUTOR", f.owner)
	e.join(f.viewer, f.space, "VIEWER", f.owner)
	return f
}

func (f *spaceFixture) changeRole(actor, target *session, role string) *request {
	return f.req("PUT", "/api/v1/spaces/%s/members/%s", f.space, target.UserID).as(actor).json(fmt.Sprintf(`{"role":%q}`, role))
}

func TestCreateSpaceMakesCreatorOwner(t *testing.T) {
	f := newSpaceFixture(t)
	res := f.req("POST", "/api/v1/spaces").as(f.viewer).json(`{"name":"  Reading list ","description":"Stuff"}`).expect(201)
	res.has("name", "Reading list").has("role", "OWNER").has("description", "Stuff")
	res.has(`permissions.#(=="DELETE_SPACE")`, "DELETE_SPACE").has(`permissions.#(=="MANAGE_MEMBERS")`, "MANAGE_MEMBERS")
	// Own spaces are listed first
	f.req("GET", "/api/v1/spaces").as(f.viewer).expect(200).has("#", 3).has("0.role", "OWNER").has("2.role", "VIEWER")
	f.req("POST", "/api/v1/spaces").as(f.owner).json(`{"name":"   "}`).expect(400)
}

func TestMembersAreListedSortedByRole(t *testing.T) {
	f := newSpaceFixture(t)
	f.req("GET", "/api/v1/spaces/%s/members", f.space).as(f.viewer).expect(200).
		has("#", 4).has("0.username", f.owner.Username).has("0.role", "OWNER").has("3.role", "VIEWER")
	f.req("GET", "/api/v1/spaces/%s/members", f.space).as(f.register()).expect(403)
	f.req("GET", "/api/v1/spaces/%s", "00000000-0000-0000-0000-000000000000").as(f.owner).expect(404)
}

func TestRolePermissions(t *testing.T) {
	f := newSpaceFixture(t)
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.contributor).json(`{"name":"Hijacked"}`).expect(403)
	f.changeRole(f.contributor, f.viewer, "CONTRIBUTOR").expect(403)
	f.createCollection(f.contributor, f.space, "Allowed", "")

	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.admin).json(`{"name":"Renamed","description":"New"}`).
		expect(200).has("name", "Renamed").has("role", "ADMIN")
	f.req("DELETE", "/api/v1/spaces/%s", f.space).as(f.admin).expect(403)

	f.req("GET", "/api/v1/spaces/%s", f.space).as(f.viewer).expect(200).has("permissions", `["READ"]`)
	f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.viewer).json(`{"name":"Nope"}`).expect(403)
}

func TestAdminsManageLowerRolesOnly(t *testing.T) {
	f := newSpaceFixture(t)
	f.changeRole(f.admin, f.viewer, "CONTRIBUTOR").expect(200).has("role", "CONTRIBUTOR")
	f.changeRole(f.admin, f.viewer, "OWNER").expect(403)
	f.changeRole(f.admin, f.owner, "VIEWER").expect(403)

	f.req("DELETE", "/api/v1/spaces/%s/members/%s", f.space, f.contributor.UserID).as(f.admin).expect(204)
	f.req("GET", "/api/v1/spaces/%s", f.space).as(f.contributor).expect(403)
}

func TestOwnershipCanBeHandedOver(t *testing.T) {
	f := newSpaceFixture(t)
	f.changeRole(f.owner, f.owner, "VIEWER").expect(400)
	f.changeRole(f.owner, f.admin, "OWNER").expect(200)
	f.changeRole(f.admin, f.owner, "VIEWER").expect(200)
	f.req("POST", "/api/v1/spaces/%s/leave", f.space).as(f.owner).expect(204)
	f.changeRole(f.admin, f.admin, "VIEWER").expect(400)
}

func TestInvitesRespectTheRoleCeiling(t *testing.T) {
	f := newSpaceFixture(t)
	f.req("POST", "/api/v1/invites").as(f.contributor).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"role":"VIEWER"}`, f.space)).expect(403)
	f.invite(f.admin, f.space, "ADMIN")
	f.req("POST", "/api/v1/invites").as(f.owner).
		json(fmt.Sprintf(`{"targetType":"COLLECTION","targetId":%q}`, f.space)).expect(400)
	f.req("POST", "/api/v1/invites").as(f.owner).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"maxUses":0}`, f.space)).expect(400)
	f.req("POST", "/api/v1/invites").as(f.owner).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"expiresAt":"2000-01-01T00:00:00Z"}`, f.space)).expect(400)
}

func TestSpaceInvitesListPreviewAndRevoke(t *testing.T) {
	f := newSpaceFixture(t)
	token := f.invite(f.owner, f.space, "CONTRIBUTOR")

	f.req("GET", "/api/v1/spaces/%s/invites", f.space).as(f.admin).expect(200).
		has(fmt.Sprintf(`#(token==%q).createdBy`, token), f.owner.Username)
	f.req("GET", "/api/v1/spaces/%s/invites", f.space).as(f.viewer).expect(403)
	f.req("GET", "/api/v1/invites").as(f.owner).expect(200).has("0.token", token)

	newcomer := f.register()
	f.req("GET", "/api/v1/invites/%s", token).as(newcomer).expect(200).
		has("targetId", f.space).has("role", "CONTRIBUTOR").has("invitedBy", f.owner.Username).
		has("valid", true).has("alreadyMember", false)

	// Admins may revoke invites created by others, members without permission may not
	f.req("DELETE", "/api/v1/invites/%s", token).as(f.viewer).expect(403)
	f.req("DELETE", "/api/v1/invites/%s", token).as(f.admin).expect(204)
	f.req("POST", "/api/v1/invites/%s", token).as(newcomer).expect(404)
}

func TestInviteLimits(t *testing.T) {
	f := newSpaceFixture(t)
	token := f.req("POST", "/api/v1/invites").as(f.owner).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"role":"VIEWER","maxUses":1}`, f.space)).expect(201).str("token")
	f.req("POST", "/api/v1/invites/%s", token).as(f.register()).expect(200).has("alreadyMember", true)
	f.req("POST", "/api/v1/invites/%s", token).as(f.register()).expect(409)
	f.req("GET", "/api/v1/invites/%s", token).as(f.owner).expect(200).has("valid", false)

	expiring := f.req("POST", "/api/v1/invites").as(f.owner).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"expiresAt":%q}`, f.space,
			time.Now().Add(time.Hour).UTC().Format(time.RFC3339))).expect(201)
	expiring.has("role", "VIEWER")
	if expiring.get("maxUses").Exists() {
		t.Fatalf("unlimited invites have no maxUses: %s", expiring.Body)
	}
}

func TestDeletingASpaceRemovesItsInvites(t *testing.T) {
	f := newSpaceFixture(t)
	token := f.invite(f.owner, f.space, "VIEWER")
	f.req("DELETE", "/api/v1/spaces/%s", f.space).as(f.owner).expect(204)
	f.req("GET", "/api/v1/invites/%s", token).as(f.owner).expect(404)
}

func TestChoosingTheStartSpace(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	first := e.firstSpace(owner)
	other := e.req("POST", "/api/v1/spaces").as(owner).json(`{"name":"Other"}`).expect(201).str("id")
	e.join(member, other, "VIEWER", owner)

	e.req("GET", "/api/v1/users/me").as(member).expect(200)
	require.False(t, e.req("GET", "/api/v1/users/me").as(member).expect(200).get("startSpaceId").Exists())
	e.req("PUT", "/api/v1/users/me/start-space").as(member).json(fmt.Sprintf(`{"spaceId":%q}`, other)).
		expect(200).has("startSpaceId", other)
	// Only spaces the user is a member of
	e.req("PUT", "/api/v1/users/me/start-space").as(member).json(fmt.Sprintf(`{"spaceId":%q}`, first)).expect(400)

	// Leaving the space unsets it
	e.req("POST", "/api/v1/spaces/%s/leave", other).as(member).expect(204)
	require.False(t, e.req("GET", "/api/v1/users/me").as(member).expect(200).get("startSpaceId").Exists())

	// Deleting the space unsets it, back to the last visited one with null
	e.req("PUT", "/api/v1/users/me/start-space").as(owner).json(fmt.Sprintf(`{"spaceId":%q}`, other)).expect(200)
	e.req("DELETE", "/api/v1/spaces/%s", other).as(owner).expect(204)
	require.False(t, e.req("GET", "/api/v1/users/me").as(owner).expect(200).get("startSpaceId").Exists())
	e.req("PUT", "/api/v1/users/me/start-space").as(owner).json(fmt.Sprintf(`{"spaceId":%q}`, first)).expect(200)
	e.req("PUT", "/api/v1/users/me/start-space").as(owner).json(`{"spaceId":null}`).expect(200)
	require.False(t, e.req("GET", "/api/v1/users/me").as(owner).expect(200).get("startSpaceId").Exists())
}
