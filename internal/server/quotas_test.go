package server_test

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math/rand/v2"
	"testing"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

func withQuotas(quotas config.Quotas) func(*config.Config) {
	return func(cfg *config.Config) { cfg.Quotas = quotas }
}

func TestLinkQuotaCountsTheTrashUntilItIsEmptied(t *testing.T) {
	e := newEnv(t, withQuotas(config.Quotas{LinksPerUser: 2}))
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	first := e.createBookmark(user, collection)
	e.createBookmark(user, collection)
	res := e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).json(`{"url":"127.0.0.1/third"}`).expect(403)
	require.Contains(t, res.str("detail"), "at most 2 links")

	e.req("DELETE", "/api/v1/bookmarks/%s", first).as(user).expect(204)
	e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).json(`{"url":"127.0.0.1/third"}`).expect(403)
	e.req("POST", "/api/v1/spaces/%s/trash/purge", space).as(user).json(`{"all":true}`).expect(204)
	e.createBookmark(user, collection)

	// The quota is per user: a member of the same space has their own
	member := e.register()
	e.join(member, space, "CONTRIBUTOR", user)
	e.createBookmark(member, collection)
}

func TestImportsStayWithinTheQuotas(t *testing.T) {
	e := newEnv(t, withQuotas(config.Quotas{LinksPerUser: 3, CollectionsPerSpace: 2}))
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	file := func(links, children int) string {
		node := `{"name":"Imported","links":[`
		for i := range links {
			if i > 0 {
				node += ","
			}
			node += fmt.Sprintf(`{"url":"example.org/%d"}`, i)
		}
		node += `],"collections":[`
		for i := range children {
			if i > 0 {
				node += ","
			}
			node += fmt.Sprintf(`{"name":"Child %d"}`, i)
		}
		return `{"format":"ragbag-collection","format_version":1,"collection":` + node + `]}}`
	}
	importFile := func(body string, preview bool) *response {
		return e.req("POST", "/api/v1/collections/%s/import", collection).as(user).query("preview", fmt.Sprint(preview)).json(body).send()
	}

	// The preview already tells, and nothing is kept
	require.Equal(t, 403, importFile(file(4, 0), true).Status)
	require.Equal(t, 403, importFile(file(4, 0), false).Status)
	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(user).expect(200).has("#", 0)
	// The file's children become sub-collections of "Links": two of them would be one too many
	require.Equal(t, 403, importFile(file(0, 2), false).Status)
	require.Equal(t, 200, importFile(file(3, 1), false).Status)
	e.req("POST", "/api/v1/spaces/%s/collections", space).as(user).json(`{"name":"One more"}`).expect(403)
}

func TestSpaceQuotaCountsOwnedSpaces(t *testing.T) {
	e := newEnv(t, withQuotas(config.Quotas{SpacesPerUser: 2}))
	user := e.register() // comes with its first space
	e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Second"}`).expect(201)
	res := e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Third"}`).expect(403)
	require.Contains(t, res.str("detail"), "at most 2 spaces")
	// Joining other spaces is not limited
	owner := e.register()
	e.join(user, e.firstSpace(owner), "VIEWER", owner)
}

func TestUploadQuota(t *testing.T) {
	e := newEnv(t, withQuotas(config.Quotas{UploadMBPerUser: 1}))
	user := e.register()
	collection := e.createCollection(user, e.firstSpace(user), "Links", "")
	bookmark := e.createBookmark(user, collection)
	other := e.createBookmark(user, collection)
	// Noise stays large as JPEG (about 600 KB at 900 px): one fits, two do not
	photo := noiseImage(t, 900, 900)
	e.req("PUT", "/api/v1/bookmarks/%s/image", bookmark).as(user).multipart(map[string]any{"file": photo}).expect(200)
	res := e.req("PUT", "/api/v1/bookmarks/%s/image", other).as(user).multipart(map[string]any{"file": photo}).expect(403)
	require.Contains(t, res.str("detail"), "1 MB")
	// Replacing an image only counts the new one
	e.req("PUT", "/api/v1/bookmarks/%s/image", bookmark).as(user).multipart(map[string]any{"file": photo}).expect(200)
}

func TestNoQuotasByDefault(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	for i := range 3 {
		e.req("POST", "/api/v1/spaces").as(user).json(fmt.Sprintf(`{"name":"Space %d"}`, i)).expect(201)
	}
}

// noiseImage is an image that hardly compresses.
func noiseImage(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	random := rand.New(rand.NewPCG(1, 2))
	for i := range img.Pix {
		img.Pix[i] = byte(random.IntN(256))
		if i%4 == 3 {
			img.Pix[i] = 255
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}
