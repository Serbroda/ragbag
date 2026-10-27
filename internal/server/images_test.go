package server_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

func pngImage(t *testing.T, width, height int) []byte {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, width, height))))
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
