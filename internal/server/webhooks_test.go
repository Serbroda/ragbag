package server_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

const discordURL = "https://discord.com/api/webhooks/123456/secret-token"

// discord stands in for Discord: it records what is posted and answers with its status.
type discord struct {
	mu       sync.Mutex
	status   int
	messages []string
	urls     []string
}

func (d *discord) send(_ context.Context, address string, payload []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.urls = append(d.urls, address)
	d.messages = append(d.messages, string(payload))
	return d.status, nil
}

// answer sets the status of the following posts (Discord answers 204, or 404 for a deleted webhook).
func (d *discord) answer(status int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.status = status
}

func (d *discord) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.messages)
}

// await waits for the n-th message (they are posted in the background).
func (d *discord) await(t *testing.T, n int) string {
	t.Helper()
	require.Eventually(t, func() bool { return d.count() >= n }, 3*time.Second, 10*time.Millisecond, "no message %d", n)
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.messages[n-1]
}

// never asserts that nothing more is posted.
func (d *discord) never(t *testing.T, n int) {
	t.Helper()
	time.Sleep(150 * time.Millisecond)
	require.Equal(t, n, d.count())
}

type webhookFixture struct {
	*env
	discord           *discord
	owner             *session
	space, collection string
}

func newWebhookFixture(t *testing.T) *webhookFixture {
	e := newEnv(t)
	f := &webhookFixture{env: e, discord: &discord{status: 204}, owner: e.register()}
	e.srv.UseWebhookSender(f.discord.send)
	f.space = e.firstSpace(f.owner)
	f.collection = e.createCollection(f.owner, f.space, "News", "")
	return f
}

func (f *webhookFixture) set(s *session, url string) *request {
	return f.req("PUT", "/api/v1/spaces/%s/webhook", f.space).as(s).json(fmt.Sprintf(`{"url":%q}`, url))
}

func (f *webhookFixture) add(collection, title string) {
	f.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(f.owner).
		json(fmt.Sprintf(`{"url":"http://127.0.0.1/%s","title":%q,"description":"See [here](http://evil.example) @everyone"}`, title[:3], title)).
		expect(200)
}

func TestNewLinksArePostedToTheDiscordWebhookOfTheSpace(t *testing.T) {
	f := newWebhookFixture(t)
	// Without a webhook nothing is posted
	f.add(f.collection, "Before")
	f.discord.never(t, 0)

	f.set(f.owner, " "+discordURL+"?wait=true ").expect(200).has("url", discordURL)
	f.req("GET", "/api/v1/spaces/%s/webhook", f.space).as(f.owner).expect(200).has("url", discordURL)
	f.add(f.collection, "Patch notes")
	message := f.discord.await(t, 1)
	require.Equal(t, discordURL, f.discord.urls[0])
	embed := gjson.Get(message, "embeds.0")
	require.Equal(t, "Patch notes", embed.Get("title").String())
	require.Equal(t, "http://127.0.0.1/Pat", embed.Get("url").String())
	require.Equal(t, f.owner.Username+" added a link", embed.Get("author.name").String())
	require.Contains(t, embed.Get("footer.text").String(), "› News")
	// Text of others is neither Markdown nor a mention
	require.Contains(t, embed.Get("description").String(), `\[here\]\(http://evil.example\)`)
	require.Equal(t, "[]", gjson.Get(message, "allowed_mentions.parse").Raw)

	f.req("DELETE", "/api/v1/spaces/%s/webhook", f.space).as(f.owner).expect(204)
	f.add(f.collection, "After")
	f.discord.never(t, 1)
}

func TestPrivateCollectionsAndImportsAreNeverPosted(t *testing.T) {
	f := newWebhookFixture(t)
	f.set(f.owner, discordURL).expect(200)
	secret := f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(`{"name":"Gifts","restricted":true}`).expect(200).str("id")
	below := f.createCollection(f.owner, f.space, "For Lena", secret)
	f.add(secret, "Secret gift")
	f.add(below, "Another gift")
	f.req("POST", "/api/v1/collections/%s/import", f.collection).as(f.owner).json(`{"format":"ragbag-collection",
		"format_version":1,"collection":{"name":"x","links":[{"url":"https://example.org/imported"}]}}`).expect(200).
		has("bookmarksImported", 1)
	f.discord.never(t, 0)
	f.add(f.collection, "Public news")
	require.Contains(t, f.discord.await(t, 1), "Public news")
}

func TestOnlyDiscordAddressesAndOnlySpaceAdminsSetTheWebhook(t *testing.T) {
	f := newWebhookFixture(t)
	for _, address := range []string{
		"http://discord.com/api/webhooks/1/x", "https://example.org/api/webhooks/1/x", "https://discord.com.evil.example/api/webhooks/1/x",
		"https://discord.com/api/users/1", "https://discord.com/api/webhooks/1", "https://user@discord.com/api/webhooks/1/x",
		"http://127.0.0.1:8080/api/webhooks/1/x", "",
	} {
		require.Equal(t, 400, f.set(f.owner, address).send().Status, address)
	}
	f.set(f.owner, discordURL).expect(200)

	contributor := f.register()
	f.join(contributor, f.space, "CONTRIBUTOR", f.owner)
	f.req("GET", "/api/v1/spaces/%s/webhook", f.space).as(contributor).expect(403)
	f.set(contributor, discordURL).expect(403)
	f.req("DELETE", "/api/v1/spaces/%s/webhook", f.space).as(contributor).expect(403)
	f.req("POST", "/api/v1/spaces/%s/webhook/test", f.space).as(contributor).expect(403)
	// The address is in nothing else members get
	require.NotContains(t, f.req("GET", "/api/v1/spaces/%s", f.space).as(contributor).expect(200).Body, "webhook")
	// API tokens never reach it, not even the owner's
	token, _ := f.apiToken(f.owner, "WRITE")
	f.req("GET", "/api/v1/spaces/%s/webhook", f.space).bearer(token).expect(403)
	f.req("PUT", "/api/v1/spaces/%s/webhook", f.space).bearer(token).json(fmt.Sprintf(`{"url":%q}`, discordURL)).expect(403)
	// … but links saved with one are announced like any other
	f.req("POST", "/api/v1/collections/%s/bookmarks", f.collection).bearer(token).json(`{"url":"http://127.0.0.1/api","title":"By script"}`).expect(200)
	require.Contains(t, f.discord.await(t, 1), "By script")
}

func TestAWebhookDiscordNoLongerKnowsStopsUntilItWorksAgain(t *testing.T) {
	f := newWebhookFixture(t)
	f.req("POST", "/api/v1/spaces/%s/webhook/test", f.space).as(f.owner).expect(400)
	f.set(f.owner, discordURL).expect(200)
	f.req("POST", "/api/v1/spaces/%s/webhook/test", f.space).as(f.owner).expect(204)
	require.Contains(t, f.discord.await(t, 1), "It works")

	// Deleted in Discord: the first link notices, later ones are not even tried
	f.discord.answer(404)
	f.add(f.collection, "Lost")
	f.discord.await(t, 2)
	require.Eventually(t, func() bool {
		return f.req("GET", "/api/v1/spaces/%s/webhook", f.space).as(f.owner).expect(200).get("failedAt").Exists()
	}, 3*time.Second, 20*time.Millisecond)
	f.add(f.collection, "Not tried")
	f.discord.never(t, 2)
	require.Contains(t, f.req("POST", "/api/v1/spaces/%s/webhook/test", f.space).as(f.owner).expect(400).str("detail"), "deleted")

	// A test that works (or a new address) turns it on again
	f.discord.answer(204)
	f.req("POST", "/api/v1/spaces/%s/webhook/test", f.space).as(f.owner).expect(204)
	require.False(t, f.req("GET", "/api/v1/spaces/%s/webhook", f.space).as(f.owner).expect(200).get("failedAt").Exists())
	f.add(f.collection, "Back again")
	require.Contains(t, f.discord.await(t, 5), "Back again")
}
