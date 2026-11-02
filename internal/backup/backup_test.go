package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/db"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func openCurrent(t *testing.T) *sqlx.DB {
	database, _, err := db.Open(context.Background(), "sqlite:"+filepath.Join(t.TempDir(), "ragbag.db"))
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	return database
}

func writeBackup(t *testing.T, database *sqlx.DB) string {
	path := filepath.Join(t.TempDir(), "backup.json.gz")
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()
	require.NoError(t, Write(context.Background(), database, db.SQLite, file, "test"))
	return path
}

// A new table must be added to the backup (or to Skipped), otherwise moving an instance loses it.
func TestEveryTableIsInTheBackup(t *testing.T) {
	var names []string
	require.NoError(t, openCurrent(t).Select(&names,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`))
	known := append(Tables(), Skipped...)
	for _, name := range names {
		require.Contains(t, known, name, "table %s is neither backed up nor skipped", name)
	}
	require.Len(t, known, len(names))
}

func TestOlderBackupsAreMigrated(t *testing.T) {
	ctx := context.Background()
	old, err := db.OpenSQLiteAt(ctx, filepath.Join(t.TempDir(), "old.db"), 10)
	require.NoError(t, err)
	defer old.Close()
	created := time.Date(2025, 3, 4, 5, 6, 7, 123456000, time.UTC)
	_, err = old.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ('u1', 'lena', 'lena@example.com', 'hash', ?, ?)`, created, created)
	require.NoError(t, err)
	_, err = old.Exec(`INSERT INTO spaces (id, name, created_by, created_at, updated_at) VALUES ('s1', 'Home', 'u1', ?, ?)`,
		created, created)
	require.NoError(t, err)
	_, err = old.Exec(`UPDATE users SET start_space_id = 's1'`)
	require.NoError(t, err)
	path := writeBackup(t, old)

	target := openCurrent(t)
	summary, err := Restore(ctx, target, db.SQLite, path, false)
	require.NoError(t, err)
	require.EqualValues(t, 10, summary.SchemaVersion)
	require.Equal(t, 1, summary.Rows["users"])
	// Migration 11 confirmed existing accounts, which ran on the backup too
	var user struct {
		Verified   *time.Time `db:"email_verified_at"`
		StartSpace *string    `db:"start_space_id"`
	}
	require.NoError(t, target.Get(&user, `SELECT email_verified_at, start_space_id FROM users WHERE id = 'u1'`))
	require.NotNil(t, user.Verified)
	require.True(t, created.Equal(*user.Verified), "%v", user.Verified)
	require.Equal(t, "s1", *user.StartSpace)
}

func TestRestoreNeedsAnEmptyInstanceAndDryRunsKeepNothing(t *testing.T) {
	ctx := context.Background()
	source := openCurrent(t)
	at := time.Now().UTC()
	_, err := source.Exec(`INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES ('u1', 'lena', 'lena@example.com', 'hash', ?, ?)`, at, at)
	require.NoError(t, err)
	_, err = source.Exec(`INSERT INTO spaces (id, name, created_by, created_at, updated_at) VALUES ('s1', 'Home', 'u1', ?, ?)`, at, at)
	require.NoError(t, err)
	// Children listed before their parents still go in
	for _, c := range [][]any{{"c3", "c2"}, {"c2", "c1"}, {"c1", nil}} {
		_, err = source.Exec(`INSERT INTO collections (id, space_id, parent_id, name, created_by, created_at, updated_at)
			VALUES (?, 's1', NULL, ?, 'u1', ?, ?)`, c[0], c[0], at, at)
		require.NoError(t, err)
	}
	for _, c := range [][]any{{"c3", "c2"}, {"c2", "c1"}} {
		_, err = source.Exec(`UPDATE collections SET parent_id = ? WHERE id = ?`, c[1], c[0])
		require.NoError(t, err)
	}
	path := writeBackup(t, source)

	// The setup admin is replaced
	target := openCurrent(t)
	_, err = target.Exec(`INSERT INTO users (id, username, email, password_hash, setup_admin, created_at, updated_at)
		VALUES ('a1', 'admin', 'admin@localhost', 'hash', 1, ?, ?)`, at, at)
	require.NoError(t, err)
	summary, err := Restore(ctx, target, db.SQLite, path, true)
	require.NoError(t, err)
	require.Equal(t, 3, summary.Rows["collections"])
	var names []string
	require.NoError(t, target.Select(&names, `SELECT username FROM users`))
	require.Equal(t, []string{"admin"}, names)

	_, err = Restore(ctx, target, db.SQLite, path, false)
	require.NoError(t, err)
	require.NoError(t, target.Select(&names, `SELECT username FROM users`))
	require.Equal(t, []string{"lena"}, names)

	// Now there is data
	_, err = Restore(ctx, target, db.SQLite, path, false)
	require.ErrorIs(t, err, ErrNotEmpty)

	// The same rows again
	again := writeBackup(t, target)
	require.Equal(t, tableRows(t, path), tableRows(t, again))
}

func TestOtherFilesAreRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "export.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"format":"ragbag-collection","format_version":2,"collection":{}}`), 0o600))
	_, err := Restore(context.Background(), openCurrent(t), db.SQLite, path, true)
	require.ErrorIs(t, err, errInvalid)
	require.NoError(t, os.WriteFile(path, []byte(`not json`), 0o600))
	_, err = Restore(context.Background(), openCurrent(t), db.SQLite, path, true)
	require.ErrorIs(t, err, errInvalid)
}

// tableRows reads a backup into table name → rows (sorted, the order is not part of it).
func tableRows(t *testing.T, path string) map[string][]string {
	r, err := open(path)
	require.NoError(t, err)
	defer r.close()
	result := map[string][]string{}
	for {
		name, _, err := r.nextTable()
		if err != nil {
			break
		}
		require.NoError(t, r.eachRow(func(row []any) error {
			result[name] = append(result[name], fmt.Sprint(row))
			return nil
		}))
		slices.Sort(result[name])
	}
	return result
}
