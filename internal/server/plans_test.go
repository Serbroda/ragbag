package server_test

import (
	"fmt"
	"testing"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

func withPlans(free, pro config.Quotas) func(*config.Config) {
	return func(cfg *config.Config) {
		cfg.Quotas = free
		cfg.Plans = config.Plans{Enabled: true, Pro: pro}
	}
}

func TestWithoutPlansEverybodyIsTheSame(t *testing.T) {
	e := newEnv(t, withQuotas(config.Quotas{SpacesPerUser: 1}))
	features := e.req("GET", "/api/app/config/features").expect(200).has("plans", false)
	require.False(t, features.get("support_url").Exists())
	user := e.register()
	e.req("GET", "/api/v1/users/me/plan").as(user).expect(404)
	detail := e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Second"}`).expect(403).str("detail")
	require.NotContains(t, detail, "Pro")
	admin := e.login("admin", "test123")
	e.req("PUT", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(404)
	e.req("GET", "/api/v1/admin/users").as(admin).expect(200)
	require.False(t, e.req("GET", "/api/v1/admin/users").as(admin).expect(200).get("0.plan").Exists())
}

func TestFreeAccountsOwnTwoSpacesAndAdminsGrantPro(t *testing.T) {
	e := newEnv(t, withPlans(config.Quotas{SpacesPerUser: 2, UploadMBPerUser: 50}, config.Quotas{UploadMBPerUser: 1000}))
	e.req("GET", "/api/app/config/features").expect(200).has("plans", true)
	user := e.register()
	// The first space of the account counts
	e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Second"}`).expect(201)
	detail := e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Third"}`).expect(403).str("detail")
	require.Contains(t, detail, "at most 2 spaces")
	require.Contains(t, detail, "Pro has more")
	e.req("GET", "/api/v1/users/me/plan").as(user).expect(200).has("plan", "FREE").has("usage.spaces", 2).
		has("free.spacesPerUser", 2).has("pro.spacesPerUser", 0).has("pro.uploadMbPerUser", 1000)

	admin := e.login("admin", "test123")
	e.req("PUT", "/api/v1/admin/users/%s/pro", user.UserID).as(e.register()).expect(403)
	e.req("PUT", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(200).has("plan", "PRO").has("proSource", "GRANTED")
	e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Third"}`).expect(201)
	e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Fourth"}`).expect(201)
	e.req("GET", "/api/v1/users/me/plan").as(user).expect(200).has("plan", "PRO").has("proSource", "GRANTED")
	e.req("GET", "/api/v1/admin/overview").as(admin).expect(200).has("grantedPro", 1).
		has("settings.plans", true).has("settings.proQuotas.uploadMbPerUser", 1000)

	// Back to Free: the spaces stay, new ones are refused
	e.req("DELETE", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(200).has("plan", "FREE")
	e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Fifth"}`).expect(403)
	require.Equal(t, int64(4), e.req("GET", "/api/v1/users/me/plan").as(user).expect(200).get("usage.spaces").Int())

	// Instance admins are Pro
	e.req("GET", "/api/v1/users/me/plan").as(admin).expect(200).has("plan", "PRO").has("proSource", "ADMIN")
	for i := range 3 {
		e.req("POST", "/api/v1/spaces").as(admin).json(fmt.Sprintf(`{"name":"Admin %d"}`, i)).expect(201)
	}
}

func TestSpacesOfProOwnersHaveProLimits(t *testing.T) {
	e := newEnv(t, withPlans(config.Quotas{CollectionsPerSpace: 1}, config.Quotas{}))
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(member, space, "CONTRIBUTOR", owner)
	e.createCollection(owner, space, "First", "")
	e.req("POST", "/api/v1/spaces/%s/collections", space).as(member).json(`{"name":"Second"}`).expect(403)

	admin := e.login("admin", "test123")
	e.req("PUT", "/api/v1/admin/users/%s/pro", owner.UserID).as(admin).expect(200)
	// Members of the space profit, whatever their own plan
	e.createCollection(member, space, "Second", "")
	e.createCollection(member, space, "Third", "")
	// Their own spaces stay Free
	own := e.firstSpace(member)
	e.createCollection(member, own, "Own", "")
	e.req("POST", "/api/v1/spaces/%s/collections", own).as(member).json(`{"name":"Too many"}`).expect(403)
}

func TestTheSupportLinkIsAFeatureFlag(t *testing.T) {
	e := newEnv(t, func(cfg *config.Config) { cfg.SupportURL = "https://ko-fi.com/example" })
	e.req("GET", "/api/app/config/features").expect(200).has("support_url", "https://ko-fi.com/example")
	admin := e.login("admin", "test123")
	e.req("GET", "/api/v1/admin/overview").as(admin).expect(200).has("settings.supportUrl", "https://ko-fi.com/example")
}
