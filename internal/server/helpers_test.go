package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/db"
	"github.com/Serbroda/ragbag/internal/server"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) {
	// The server logs every refused metadata fetch; keep test output readable
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

// env is a running server with its own empty database: a SQLite file, or a fresh PostgreSQL schema when
// RAGBAG_TEST_POSTGRES is set (e.g. postgres://postgres:test@localhost:15432/ragbag?sslmode=disable).
type env struct {
	t   *testing.T
	url string
	srv *server.Server
	db  *sqlx.DB
}

type session struct {
	UserID, Username, Token string
	Refresh                 *http.Cookie
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	databaseURL := "sqlite:" + t.TempDir() + "/test.db"
	if base := os.Getenv("RAGBAG_TEST_POSTGRES"); base != "" {
		databaseURL = postgresSchema(t, base)
	}
	database, dialect, err := db.Open(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })

	cfg := config.Config{
		JWTSecret:        []byte("test-secret-test-secret-test-secret-123"),
		JWTIssuer:        "ragbag",
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  time.Hour,
		RegistrationOpen: true,
		AdminUsers:       []string{"admin"},
	}
	srv := server.New(cfg, store.New(database, dialect), "test")
	require.NoError(t, srv.SeedDemo(ctx))
	// A stand-in for the frontend build: link previews are written into its head
	frontend := fstest.MapFS{"index.html": {Data: []byte("<html><head><title>Ragbag</title></head><body></body></html>")}}
	httpServer := httptest.NewServer(srv.Handler(frontend))
	t.Cleanup(httpServer.Close)
	return &env{t: t, url: httpServer.URL, srv: srv, db: database}
}

func postgresSchema(t *testing.T, base string) string {
	schema := "t_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	admin, err := sqlx.Open("pgx", base)
	require.NoError(t, err)
	_, err = admin.Exec("CREATE SCHEMA " + schema)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		admin.Close()
	})
	separator := "?"
	if strings.Contains(base, "?") {
		separator = "&"
	}
	return base + separator + "search_path=" + schema
}

// --- requests ---

type request struct {
	e       *env
	method  string
	path    string
	token   string
	body    io.Reader
	ctype   string
	cookies []*http.Cookie
	headers map[string]string
}

func (e *env) req(method, path string, args ...any) *request {
	return &request{e: e, method: method, path: fmt.Sprintf(path, args...), headers: map[string]string{}}
}

func (r *request) as(s *session) *request { r.token = s.Token; return r }

func (r *request) bearer(token string) *request { r.token = token; return r }

func (r *request) cookie(c *http.Cookie) *request { r.cookies = append(r.cookies, c); return r }

func (r *request) header(key, value string) *request { r.headers[key] = value; return r }

func (r *request) json(body string) *request {
	r.body, r.ctype = strings.NewReader(body), "application/json"
	return r
}

func (r *request) query(key, value string) *request {
	separator := "?"
	if strings.Contains(r.path, "?") {
		separator = "&"
	}
	r.path += separator + key + "=" + url.QueryEscape(value)
	return r
}

// multipart sends form fields; a field named "file" is sent as a file upload.
func (r *request) multipart(fields map[string]any) *request {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for name, value := range fields {
		switch v := value.(type) {
		case []byte:
			part, _ := writer.CreateFormFile(name, "upload.bin")
			_, _ = part.Write(v)
		case string:
			_ = writer.WriteField(name, v)
		}
	}
	_ = writer.Close()
	r.body, r.ctype = &buf, writer.FormDataContentType()
	return r
}

type response struct {
	t      *testing.T
	Status int
	Body   string
	Header http.Header
}

func (r *request) send() *response {
	r.e.t.Helper()
	httpReq, err := http.NewRequest(r.method, r.e.url+r.path, r.body)
	require.NoError(r.e.t, err)
	if r.ctype != "" {
		httpReq.Header.Set("Content-Type", r.ctype)
	}
	if r.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.token)
	}
	for key, value := range r.headers {
		httpReq.Header.Set(key, value)
	}
	for _, c := range r.cookies {
		httpReq.AddCookie(c)
	}
	res, err := http.DefaultClient.Do(httpReq)
	require.NoError(r.e.t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(r.e.t, err)
	return &response{t: r.e.t, Status: res.StatusCode, Body: string(body), Header: res.Header}
}

// expect asserts the status and returns the response for further checks.
func (r *request) expect(status int) *response {
	r.e.t.Helper()
	res := r.send()
	require.Equal(r.e.t, status, res.Status, "%s %s: %s", r.method, r.path, res.Body)
	return res
}

// get reads a gjson path, e.g. "0.id", "#" (array length), "#(token==\"x\").createdBy".
func (r *response) get(path string) gjson.Result { return gjson.Get(r.Body, path) }

func (r *response) str(path string) string {
	r.t.Helper()
	result := r.get(path)
	require.True(r.t, result.Exists(), "missing %q in %s", path, r.Body)
	return result.String()
}

func (r *response) has(path string, expected any) *response {
	r.t.Helper()
	result := r.get(path)
	require.True(r.t, result.Exists(), "missing %q in %s", path, r.Body)
	require.Equal(r.t, fmt.Sprint(expected), result.String(), "path %q in %s", path, r.Body)
	return r
}

func (r *response) cookieNamed(name string) *http.Cookie {
	for _, c := range (&http.Response{Header: r.Header}).Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// --- fixtures ---

func (e *env) register() *session {
	e.t.Helper()
	username := "u" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	e.req("POST", "/api/auth/register").
		json(fmt.Sprintf(`{"username":%q,"email":"%s@example.com","password":"secret123"}`, username, username)).
		expect(200)
	s := e.login(username, "secret123")
	s.UserID = e.req("GET", "/api/v1/users/me").as(s).expect(200).str("id")
	return s
}

func (e *env) login(username, password string) *session {
	e.t.Helper()
	res := e.req("POST", "/api/auth/login").json(fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)).expect(200)
	refresh := res.cookieNamed("refresh_token")
	require.NotNil(e.t, refresh)
	return &session{Username: username, Token: res.str("access_token"), Refresh: &http.Cookie{Name: refresh.Name, Value: refresh.Value}}
}

func (e *env) firstSpace(s *session) string {
	return e.req("GET", "/api/v1/spaces").as(s).expect(200).str("0.id")
}

func (e *env) createCollection(s *session, spaceID, name, parentID string) string {
	e.t.Helper()
	parent := ""
	if parentID != "" {
		parent = fmt.Sprintf(`,"parentId":%q`, parentID)
	}
	return e.req("POST", "/api/v1/spaces/%s/collections", spaceID).as(s).
		json(fmt.Sprintf(`{"name":%q%s}`, name, parent)).expect(200).str("id")
}

// createBookmark uses a loopback URL: the metadata fetcher must refuse it, so tests never touch the network.
func (e *env) createBookmark(s *session, collectionID string) string {
	e.t.Helper()
	return e.req("POST", "/api/v1/collections/%s/bookmarks", collectionID).as(s).
		json(`{"url":"http://127.0.0.1/page","title":"Example","tags":["a"]}`).expect(200).str("id")
}

func (e *env) invite(s *session, spaceID, role string) string {
	e.t.Helper()
	return e.req("POST", "/api/v1/invites").as(s).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"role":%q}`, spaceID, role)).expect(201).str("token")
}

func (e *env) join(s *session, spaceID, role string, owner *session) {
	e.t.Helper()
	e.req("POST", "/api/v1/invites/%s", e.invite(owner, spaceID, role)).as(s).expect(200)
}

var _ = json.Marshal
