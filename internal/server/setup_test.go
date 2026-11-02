package server_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

func emptyEnv(t *testing.T, options ...func(*config.Config)) *env {
	databaseURL := "sqlite:" + t.TempDir() + "/test.db"
	if base := os.Getenv("RAGBAG_TEST_POSTGRES"); base != "" {
		databaseURL = postgresSchema(t, base)
	}
	return startEnv(t, databaseURL, false, options...)
}

func (e *env) initialPassword() string {
	e.t.Helper()
	data, err := os.ReadFile(filepath.Join(e.cfg.DataDir, "initial-admin-password"))
	require.NoError(e.t, err)
	return strings.TrimSpace(string(data))
}

// setUp signs in as the setup admin and replaces the initial password.
func (e *env) setUp(username string) *session {
	e.t.Helper()
	require.NoError(e.t, e.srv.Bootstrap(context.Background()))
	initial := e.initialPassword()
	admin := e.login(username, initial)
	res := e.req("POST", "/api/auth/setup").as(admin).
		json(fmt.Sprintf(`{"currentPassword":%q,"newPassword":%q,"email":"me@example.com"}`, initial, testPassword)).expect(200)
	return &session{Username: username, Token: res.str("access_token")}
}

func TestTheFirstStartCreatesAnAdminWithARandomPassword(t *testing.T) {
	e := emptyEnv(t, func(cfg *config.Config) { cfg.AdminUsers = nil })
	ctx := context.Background()
	require.NoError(t, e.srv.Bootstrap(ctx))
	path := filepath.Join(e.cfg.DataDir, "initial-admin-password")
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	initial := e.initialPassword()
	require.GreaterOrEqual(t, len(initial), 20)
	// Only once
	require.NoError(t, e.srv.Bootstrap(ctx))
	require.Equal(t, initial, e.initialPassword())

	admin := e.login("admin", initial)
	e.req("GET", "/api/v1/users/me").as(admin).expect(200).has("mustChangePassword", true).has("instanceAdmin", true)
	// Nothing else until the password is replaced
	e.req("GET", "/api/v1/spaces").as(admin).expect(403)
	e.req("GET", "/api/v1/admin/instance").as(admin).expect(403)
	setup := func(current, next, email string) *request {
		return e.req("POST", "/api/auth/setup").as(admin).
			json(fmt.Sprintf(`{"currentPassword":%q,"newPassword":%q,"email":%q}`, current, next, email))
	}
	setup("wrong-password", testPassword, "me@example.com").expect(400)
	setup(initial, initial, "me@example.com").expect(400)
	setup(initial, testPassword, "no address").expect(400)
	res := setup(initial, testPassword, "Me@Example.com").expect(200)
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)

	// The old session is signed out, the new one may do everything
	e.req("GET", "/api/v1/users/me").as(admin).expect(401)
	admin = &session{Token: res.str("access_token")}
	me := e.req("GET", "/api/v1/users/me").as(admin).expect(200).has("email", "me@example.com")
	require.False(t, me.get("mustChangePassword").Exists())
	e.req("GET", "/api/v1/spaces").as(admin).expect(200).has("#", 0)
	e.req("GET", "/api/v1/admin/instance").as(admin).expect(200).has("empty", true)
	e.req("POST", "/api/auth/setup").as(admin).
		json(fmt.Sprintf(`{"currentPassword":%q,"newPassword":"another-secret","email":"me@example.com"}`, testPassword)).
		expect(409)
	e.login("admin", testPassword)

	// While ADMIN_USERS is empty only the setup admin is one; others are not
	user := e.register()
	e.req("GET", "/api/v1/admin/users").as(user).expect(403)
	e.req("GET", "/api/v1/admin/users").as(admin).expect(200).has("#", 2)
	e.req("GET", "/api/v1/admin/instance").as(admin).expect(200).has("empty", false)
}

func TestTheSetupAdminIsTheFirstOfAdminUsers(t *testing.T) {
	e := emptyEnv(t, func(cfg *config.Config) { cfg.AdminUsers = []string{"Danny", "lena"} })
	admin := e.setUp("danny")
	e.req("GET", "/api/v1/users/me").as(admin).expect(200).has("username", "danny").has("instanceAdmin", true)
}

func TestChangingThePasswordAlsoEndsTheSetup(t *testing.T) {
	e := emptyEnv(t)
	require.NoError(t, e.srv.Bootstrap(context.Background()))
	initial := e.initialPassword()
	admin := e.login("admin", initial)
	res := e.req("POST", "/api/auth/password").as(admin).
		json(fmt.Sprintf(`{"currentPassword":%q,"newPassword":%q}`, initial, testPassword)).expect(200)
	admin = &session{Token: res.str("access_token")}
	e.req("GET", "/api/v1/spaces").as(admin).expect(200)
	_, err := os.Stat(filepath.Join(e.cfg.DataDir, "initial-admin-password"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func (e *env) download(s *session) []byte {
	e.t.Helper()
	res := e.req("GET", "/api/v1/admin/backup").as(s).expect(200)
	require.Equal(e.t, "application/gzip", res.Header.Get("Content-Type"))
	require.Contains(e.t, res.Header.Get("Content-Disposition"), "ragbag-backup-")
	return []byte(res.Body)
}

func (e *env) restore(s *session, backup []byte, dryRun bool) *request {
	r := e.req("POST", "/api/v1/admin/backup/restore").as(s)
	if dryRun {
		r = r.query("dryRun", "true")
	}
	r.body, r.ctype = bytes.NewReader(backup), "application/octet-stream"
	return r
}

// backupRows reads a backup into table → rows as column → value, sorted; booleans as 0/1 (SQLite's).
func backupRows(t *testing.T, data []byte) map[string][]string {
	zipped, err := gzip.NewReader(bytes.NewReader(data))
	require.NoError(t, err)
	plain, err := io.ReadAll(zipped)
	require.NoError(t, err)
	var file struct {
		Tables []struct {
			Name    string   `json:"name"`
			Columns []string `json:"columns"`
			Rows    [][]any  `json:"rows"`
		} `json:"tables"`
	}
	require.NoError(t, json.Unmarshal(plain, &file))
	result := map[string][]string{}
	for _, table := range file.Tables {
		rows := []string{}
		for _, row := range table.Rows {
			values := map[string]any{}
			for i, value := range row {
				if b, ok := value.(bool); ok {
					value = 0.0
					if b {
						value = 1.0
					}
				}
				values[table.Columns[i]] = value
			}
			encoded, _ := json.Marshal(values)
			rows = append(rows, string(encoded))
		}
		slices.Sort(rows)
		result[table.Name] = rows
	}
	return result
}

// Moving an instance, e.g. from SQLite at home to PostgreSQL on a server (when RAGBAG_TEST_POSTGRES is set).
func TestMovingAnInstanceWithABackup(t *testing.T) {
	source := startEnv(t, "sqlite:"+t.TempDir()+"/home.db", true)
	admin := source.login("admin", "test123")
	lena, bob := source.register(), source.register()
	space := source.firstSpace(lena)
	parent := source.createCollection(lena, space, "Reading", "")
	child := source.createCollection(lena, space, "Later", parent)
	bookmark := source.createBookmark(lena, child)
	source.createBookmark(lena, parent)
	trashed := source.createBookmark(lena, parent)
	source.req("DELETE", "/api/v1/bookmarks/%s", trashed).as(lena).expect(204)
	source.join(bob, space, "CONTRIBUTOR", lena)
	source.req("PUT", "/api/v1/bookmarks/%s/reactions", bookmark).as(bob).json(`{"emoji":"👍","reacted":true}`).expect(200)
	source.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(bob).expect(204)
	share := source.req("PUT", "/api/v1/collections/%s/share", parent).as(lena).expect(200).str("token")
	avatar := source.req("PUT", "/api/v1/users/me/avatar").as(lena).
		multipart(map[string]any{"file": pngImage(t, 300, 300)}).expect(200).str("avatarId")
	source.req("PUT", "/api/v1/users/me/start-space").as(lena).json(fmt.Sprintf(`{"spaceId":%q}`, space)).expect(200)
	source.setPin(lena, "1234")

	// Only instance admins
	source.req("GET", "/api/v1/admin/backup").as(lena).expect(403)
	backup := source.download(admin)

	targetURL := "sqlite:" + t.TempDir() + "/server.db"
	if base := os.Getenv("RAGBAG_TEST_POSTGRES"); base != "" {
		targetURL = postgresSchema(t, base)
	}
	target := startEnv(t, targetURL, false)
	setupAdmin := target.setUp("admin")
	target.restore(lena, backup, true).expect(401)
	target.req("POST", "/api/v1/admin/backup/restore").as(setupAdmin).json(`{"format":"ragbag-collection"}`).expect(400)
	summary := target.restore(setupAdmin, backup, true).expect(200)
	summary.has("appVersion", "test").has("rows.users", 3).has("rows.bookmarks", 3).has("rows.collections", 9)
	// The dry run changed nothing
	target.req("GET", "/api/v1/users/me").as(setupAdmin).expect(200).has("username", "admin")
	target.restore(setupAdmin, backup, false).expect(200)

	// Everything is there, with the same ids: sessions are not (another secret), the passwords are
	target.req("GET", "/api/v1/users/me").as(setupAdmin).expect(401)
	lena = target.login(lena.Username, testPassword)
	target.req("GET", "/api/v1/users/me").as(lena).expect(200).has("startSpaceId", space).has("avatarId", avatar).
		has("hasPin", true)
	target.req("GET", "/api/v1/collections/%s/bookmarks", child).as(lena).expect(200).has("0.id", bookmark)
	target.req("GET", "/api/v1/spaces/%s/trash", space).as(lena).expect(200).has("bookmarks.#", 1)
	target.req("GET", "/api/v1/shares/%s", share).expect(200).has("name", "Reading")
	res := target.req("GET", "/api/v1/images/%s", avatar).expect(200)
	require.Equal(t, "image/png", res.Header.Get("Content-Type"))
	bob = target.login(bob.Username, testPassword)
	target.req("GET", "/api/v1/spaces/%s/favorites", space).as(bob).expect(200).has("#", 1)
	admin = target.login("admin", "test123")

	// Row by row the same, whatever the database
	require.Equal(t, backupRows(t, backup), backupRows(t, target.download(admin)))

	// Never over existing data
	target.restore(admin, backup, true).expect(409)
	require.NotEqual(t, http.StatusOK, target.restore(admin, backup, false).send().Status)
}
