package server_test

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// lockedBuffer collects log output; the test server logs from its own goroutines.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func captureLogs(t *testing.T) *lockedBuffer {
	logs := &lockedBuffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return logs
}

func TestRequestsAndLoginsAreLoggedWithoutSecrets(t *testing.T) {
	e := newEnv(t)
	logs := captureLogs(t)
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	token := e.req("PUT", "/api/v1/collections/%s/share", collection).as(user).expect(200).str("token")

	e.req("POST", "/api/auth/login").json(`{"username":"nobody","password":"wrong-password-123"}`).
		header("X-Forwarded-For", "203.0.113.7").expect(401)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(user).query("q", "my secret search").expect(200)
	e.req("GET", "/api/v1/shares/%s", token).expect(200)
	e.req("GET", "/api/v1/collections/%s", "00000000-0000-0000-0000-000000000000").as(user).expect(404)

	out := logs.String()
	// The test server is reached over loopback, a trusted proxy address, so the forwarded client IP counts
	require.Contains(t, out, `level=WARN msg="login failed" login=nobody ip=203.0.113.7`)
	require.Contains(t, out, `msg=login user=`+user.Username)
	require.Contains(t, out, "path=/api/v1/spaces/"+space+"/bookmarks/search status=200")
	require.Contains(t, out, "path=/api/v1/shares/*** status=200")
	require.Contains(t, out, "user="+user.Username)
	require.Contains(t, out, "status=404")
	for _, secret := range []string{token, "my secret search", "my+secret", "wrong-password-123", "secret123", user.Token} {
		require.NotContains(t, out, secret)
	}
	// The frontend's files are only logged at DEBUG
	e.req("GET", "/app/").send()
	require.True(t, strings.Contains(logs.String(), "level=DEBUG msg=request method=GET path=/app/"), logs.String())
}
