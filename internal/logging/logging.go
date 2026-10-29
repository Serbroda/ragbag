// Package logging sets up the server's log output: stdout or a rotating file, text or JSON, with a level.
package logging

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	Level slog.Level
	// JSON writes one JSON object per line (for log shippers), otherwise logfmt-like text
	JSON bool
	// File is a path to log into instead of stdout; it rotates at MaxSizeMB and keeps MaxFiles compressed copies
	File      string
	MaxSizeMB int
	MaxFiles  int
}

// ParseLevel accepts debug, info, warn and error (case-insensitive).
func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "", "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error, not %q", value)
}

// Setup makes the configured logger the default and returns what has to be closed on shutdown.
func Setup(cfg Config) (io.Closer, error) {
	var out io.Writer = os.Stdout
	var closer io.Closer = io.NopCloser(nil)
	if cfg.File != "" {
		if err := checkWritable(cfg.File); err != nil {
			return nil, err
		}
		file := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxFiles,
			Compress:   true,
		}
		out, closer = file, file
	}
	options := &slog.HandlerOptions{Level: cfg.Level}
	var handler slog.Handler = slog.NewTextHandler(out, options)
	if cfg.JSON {
		handler = slog.NewJSONHandler(out, options)
	}
	slog.SetDefault(slog.New(handler))
	return closer, nil
}

// checkWritable fails at startup with a helpful message instead of silently losing every log line later.
func checkWritable(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("cannot create the directory for LOG_FILE %s: %w", path, err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		hint := ""
		if errors.Is(err, os.ErrPermission) {
			hint = " (with the systemd service use /var/log/ragbag/, which systemd creates for it)"
		}
		return fmt.Errorf("cannot write LOG_FILE %s: %w%s", path, err, hint)
	}
	return file.Close()
}
