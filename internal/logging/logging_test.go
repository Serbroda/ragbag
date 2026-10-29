package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	for value, want := range map[string]slog.Level{"": slog.LevelInfo, "DEBUG": slog.LevelDebug, "warning": slog.LevelWarn, " error ": slog.LevelError} {
		level, err := ParseLevel(value)
		require.NoError(t, err)
		require.Equal(t, want, level)
	}
	_, err := ParseLevel("verbose")
	require.Error(t, err)
}

func TestLogsIntoAFileAtTheLevel(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	path := filepath.Join(t.TempDir(), "logs", "ragbag.log")

	closer, err := Setup(Config{Level: slog.LevelWarn, JSON: true, File: path, MaxSizeMB: 1, MaxFiles: 1})
	require.NoError(t, err)
	slog.Info("hidden")
	slog.Warn("visible", "key", "value")
	require.NoError(t, closer.Close())

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotContains(t, string(content), "hidden")
	require.Contains(t, string(content), `"msg":"visible","key":"value"`)
}

func TestUnwritableLogFileFailsAtStartup(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	_, err := Setup(Config{File: filepath.Join(dir, "ragbag.log")})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "cannot write LOG_FILE"), err.Error())
}
