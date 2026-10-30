package server_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func pngImage(t *testing.T, width, height int) []byte {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))))
	return buf.Bytes()
}

// photoImage is an opaque image, like a photo.
func photoImage(t *testing.T, width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 200, G: 120, B: 40, A: 255}), image.Point{}, draw.Src)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestAvatarIsCroppedScaledAndServedPublicly(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	imageID := e.req("PUT", "/api/v1/users/me/avatar").as(user).
		multipart(map[string]any{"file": pngImage(t, 800, 400)}).expect(200).str("avatarId")

	res := e.req("GET", "/api/v1/images/%s", imageID).expect(200)
	require.Equal(t, "image/png", res.Header.Get("Content-Type"))
	require.Contains(t, res.Header.Get("Cache-Control"), "immutable")
	require.Contains(t, res.Header.Get("Content-Security-Policy"), "sandbox")
	config, err := png.DecodeConfig(bytes.NewReader([]byte(res.Body)))
	require.NoError(t, err)
	require.Equal(t, 256, config.Width)
	require.Equal(t, 256, config.Height)

	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("avatarId", imageID)
}

func TestReplacingOrRemovingTheAvatarDeletesTheOldImage(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	first := e.req("PUT", "/api/v1/users/me/avatar").as(user).multipart(map[string]any{"file": pngImage(t, 64, 64)}).expect(200).str("avatarId")
	second := e.req("PUT", "/api/v1/users/me/avatar").as(user).multipart(map[string]any{"file": pngImage(t, 32, 32)}).expect(200).str("avatarId")
	require.NotEqual(t, first, second)
	e.req("GET", "/api/v1/images/%s", first).expect(404)

	res := e.req("DELETE", "/api/v1/users/me/avatar").as(user).expect(200)
	require.False(t, res.get("avatarId").Exists())
	e.req("GET", "/api/v1/images/%s", second).expect(404)
}

func TestNonImagesAreRejected(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	e.req("PUT", "/api/v1/users/me/avatar").as(user).multipart(map[string]any{"file": []byte("<svg onload=alert(1)></svg>")}).expect(400)
	// The URL variant goes through the SSRF guard
	e.req("PUT", "/api/v1/users/me/avatar").as(user).multipart(map[string]any{"url": "http://127.0.0.1/avatar.png"}).expect(400)
	e.req("PUT", "/api/v1/users/me/avatar").as(user).multipart(map[string]any{}).expect(400)
}

func TestSpaceIconNeedsEditPermission(t *testing.T) {
	e := newEnv(t)
	owner, viewer := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(viewer, space, "VIEWER", owner)

	e.req("PUT", "/api/v1/spaces/%s/icon", space).as(viewer).multipart(map[string]any{"file": pngImage(t, 16, 16)}).expect(403)
	iconID := e.req("PUT", "/api/v1/spaces/%s/icon", space).as(owner).multipart(map[string]any{"file": pngImage(t, 16, 16)}).
		expect(200).str("iconId")
	e.req("GET", "/api/v1/spaces").as(viewer).expect(200).has(`#(id=="`+space+`").iconId`, iconID)

	res := e.req("DELETE", "/api/v1/spaces/%s/icon", space).as(owner).expect(200)
	require.False(t, res.get("iconId").Exists())
	e.req("GET", "/api/v1/images/%s", iconID).expect(404)
}

func TestOwnPreviewImagesOfLinks(t *testing.T) {
	e := newEnv(t)
	owner, viewer := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(viewer, space, "VIEWER", owner)
	collection := e.createCollection(owner, space, "Links", "")
	bookmark := e.createBookmark(owner, collection)

	// Uploads keep their aspect ratio and are scaled down, photos become JPEG
	imageID := e.req("PUT", "/api/v1/bookmarks/%s/image", bookmark).as(owner).
		multipart(map[string]any{"file": photoImage(t, 2400, 1000)}).expect(200).str("customImageId")
	res := e.req("GET", "/api/v1/images/%s", imageID).expect(200)
	require.Equal(t, "image/jpeg", res.Header.Get("Content-Type"))
	config, err := jpeg.DecodeConfig(bytes.NewReader([]byte(res.Body)))
	require.NoError(t, err)
	require.Equal(t, 1200, config.Width)
	require.Equal(t, 500, config.Height)
	e.req("PUT", "/api/v1/bookmarks/%s/image", bookmark).as(viewer).multipart(map[string]any{"file": pngImage(t, 10, 10)}).expect(403)
	// Transparency survives as PNG
	transparentID := e.req("PUT", "/api/v1/bookmarks/%s/image", bookmark).as(owner).
		multipart(map[string]any{"file": pngImage(t, 300, 100)}).expect(200).str("customImageId")
	require.Equal(t, "image/png", e.req("GET", "/api/v1/images/%s", transparentID).expect(200).Header.Get("Content-Type"))
	// The replaced upload is gone
	e.req("GET", "/api/v1/images/%s", imageID).expect(404)
	imageID = transparentID

	// Changing the link keeps the own image
	update := func(extra string) *response {
		return e.req("PUT", "/api/v1/bookmarks/%s", bookmark).as(owner).
			json(`{"url":"http://127.0.0.1/other","title":"Example","tags":["a"]` + extra + `}`).expect(200)
	}
	update("").has("customImageId", imageID)

	// An own address replaces the upload, which is deleted
	update(`,"customImageUrl":"example.org/pic.png"`).has("customImageUrl", "https://example.org/pic.png")
	require.False(t, e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(200).get("customImageId").Exists())
	e.req("GET", "/api/v1/images/%s", imageID).expect(404)
	e.req("PUT", "/api/v1/bookmarks/%s", bookmark).as(owner).
		json(`{"url":"http://127.0.0.1/other","customImageUrl":"javascript:alert(1)"}`).expect(400)

	// The public view shows what the app shows
	token := e.req("PUT", "/api/v1/collections/%s/share", collection).as(owner).expect(200).str("token")
	e.req("GET", "/api/v1/shares/%s", token).expect(200).has("bookmarks.0.ogImage", "https://example.org/pic.png")
	update(`,"imageHidden":true`).has("imageHidden", true)
	require.False(t, e.req("GET", "/api/v1/shares/%s", token).expect(200).get("bookmarks.0.ogImage").Exists())

	e.req("DELETE", "/api/v1/bookmarks/%s/image", bookmark).as(owner).expect(200).has("imageHidden", false)
	require.False(t, e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(200).get("customImageUrl").Exists())
}

func TestUploadsOfDeletedLinksAreCleanedUp(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	collection := e.createCollection(owner, e.firstSpace(owner), "Links", "")
	bookmark := e.createBookmark(owner, collection)
	imageID := e.req("PUT", "/api/v1/bookmarks/%s/image", bookmark).as(owner).
		multipart(map[string]any{"file": pngImage(t, 40, 20)}).expect(200).str("customImageId")
	space := e.firstSpace(owner)
	e.req("DELETE", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(204)

	// In the trash the image is kept for restoring; deleted for good it goes with the next cleanup
	_, err := e.db.Exec(e.db.Rebind(`UPDATE images SET created_at = ?`), time.Now().Add(-2*time.Hour).UTC())
	require.NoError(t, err)
	e.srv.PurgeOldTrash(context.Background())
	e.req("GET", "/api/v1/images/%s", imageID).expect(200)
	e.req("POST", "/api/v1/spaces/%s/trash/purge", space).as(owner).json(`{"all":true}`).expect(204)
	e.srv.PurgeOldTrash(context.Background())
	e.req("GET", "/api/v1/images/%s", imageID).expect(404)
}
