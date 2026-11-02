package server_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/health").expect(200).has("status", "ok").has("version", "test")
	e.db.Close()
	e.req("GET", "/api/health").expect(503).has("status", "unavailable")
}

func TestStoredBackups(t *testing.T) {
	e := newEnv(t, func(cfg *config.Config) {
		cfg.Backups = config.Backups{Dir: filepath.Join(cfg.DataDir, "backups"), Keep: 2, Interval: 24 * time.Hour}
	})
	admin, user := e.login("admin", "test123"), e.register()
	e.req("GET", "/api/v1/admin/backups").as(user).expect(403)
	e.req("POST", "/api/v1/admin/backups").as(user).expect(403)
	e.req("GET", "/api/v1/admin/backups").as(admin).expect(200).
		has("enabled", true).has("keep", 2).has("intervalHours", 24).has("files.#", 0)

	// Two older ones, then one by hand: only the newest two stay
	dir := filepath.Join(e.cfg.DataDir, "backups")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	for _, name := range []string{"ragbag-20250101-000000.json.gz", "ragbag-20250102-000000.json.gz"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600))
	}
	res := e.req("POST", "/api/v1/admin/backups").as(admin).expect(200).has("files.#", 2).
		has("files.1.name", "ragbag-20250102-000000.json.gz")
	name := res.str("files.0.name")
	// The next one is due an interval after the newest
	next, err := time.Parse(time.RFC3339, res.str("nextAt"))
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(24*time.Hour), next, 2*time.Minute)
	e.req("GET", "/api/v1/admin/overview").as(admin).expect(200).has("settings.backupKeep", 2)

	download := e.req("GET", "/api/v1/admin/backups/%s", name).as(admin).expect(200)
	require.Contains(t, download.Header.Get("Content-Disposition"), name)
	require.Equal(t, "application/gzip", download.Header.Get("Content-Type"))
	// A stored backup restores like a downloaded one
	target := emptyEnv(t)
	require.NoError(t, target.srv.Bootstrap(context.Background()))
	setupAdmin := target.login("admin", target.initialPassword())
	target.restore(setupAdmin, []byte(download.Body), true).expect(200).has("rows.users", 2)

	e.req("GET", "/api/v1/admin/backups/%s", "notes.txt").as(admin).expect(404)
	e.req("GET", "/api/v1/admin/backups/%s", "..%2Fragbag.db").as(admin).expect(404)
	e.req("GET", "/api/v1/admin/backups/%s", "ragbag-20250101-000000.json.gz").as(admin).expect(404)
}

func TestScheduledBackupsWaitForTheInterval(t *testing.T) {
	e := newEnv(t, func(cfg *config.Config) {
		cfg.Backups = config.Backups{Dir: filepath.Join(cfg.DataDir, "backups"), Keep: 3, Interval: 24 * time.Hour}
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { e.srv.ScheduledBackups(ctx); close(done) }()
	// Waits at least a minute after starting, and stops with the server
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done
	entries, _ := os.ReadDir(filepath.Join(e.cfg.DataDir, "backups"))
	require.Empty(t, entries)

	// Without BACKUP_KEEP nothing is scheduled
	off := newEnv(t)
	finished := make(chan struct{})
	go func() { off.srv.ScheduledBackups(context.Background()); close(finished) }()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("ScheduledBackups should return at once without BACKUP_KEEP")
	}
}

func TestAnonymizedIPsInTheLog(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	e := newEnv(t, func(cfg *config.Config) { cfg.AnonymizeIPs = true })
	e.req("POST", "/api/auth/login").json(`{"username":"admin","password":"wrong-password"}`).expect(401)
	log := buf.String()
	require.Contains(t, log, "ip=127.0.0.0")
	require.False(t, strings.Contains(log, "ip=127.0.0.1"), log)
}
