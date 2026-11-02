package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Serbroda/ragbag/internal/db"
	"github.com/stretchr/testify/require"
)

func TestBackupDirectory(t *testing.T) {
	database := openCurrent(t)
	dir := Dir{Path: filepath.Join(t.TempDir(), "backups")}
	files, err := dir.List()
	require.NoError(t, err)
	require.Empty(t, files, "a missing directory has no backups")

	created, err := dir.Create(context.Background(), database, db.SQLite, "test")
	require.NoError(t, err)
	require.Regexp(t, `^ragbag-\d{8}-\d{6}\.json\.gz$`, created.Name)
	info, err := os.Stat(filepath.Join(dir.Path, created.Name))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	path, err := dir.Open(created.Name)
	require.NoError(t, err)
	header, err := readHeader(path)
	require.NoError(t, err)
	require.Equal(t, Format, header.Format)

	// Older ones, and files that are no backups of the directory
	for _, name := range []string{"ragbag-20250101-000000.json.gz", "ragbag-20250102-000000.json.gz", "notes.txt"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir.Path, name), []byte("x"), 0o600))
	}
	files, err = dir.List()
	require.NoError(t, err)
	require.Len(t, files, 3)
	require.Equal(t, created.Name, files[0].Name, "newest first")

	removed, err := dir.Prune(2)
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	files, _ = dir.List()
	require.Equal(t, []string{created.Name, "ragbag-20250102-000000.json.gz"}, []string{files[0].Name, files[1].Name})
	_, err = os.Stat(filepath.Join(dir.Path, "notes.txt"))
	require.NoError(t, err, "other files stay")

	for _, name := range []string{"../ragbag.db", "notes.txt", "ragbag-20250101-000000.json.gz"} {
		_, err := dir.Open(name)
		require.ErrorIs(t, err, ErrNoBackup, name)
	}
}
