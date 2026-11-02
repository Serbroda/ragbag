package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/db"
	"github.com/jmoiron/sqlx"
)

// Dir keeps backup files made by the server, e.g. every night: ragbag-20261102-031500.json.gz, newest last.
type Dir struct {
	Path string
}

// File is a backup in the directory.
type File struct {
	Name      string
	Size      int64
	CreatedAt time.Time
}

const fileLayout = "20060102-150405"

var fileName = regexp.MustCompile(`^ragbag-\d{8}-\d{6}\.json\.gz$`)

// ErrNoBackup: no such file in the directory (or a name that is none of its backups).
var ErrNoBackup = errors.New("no such backup")

// Create writes a backup of the database into the directory; a half written one never shows up there.
func (d Dir) Create(ctx context.Context, database *sqlx.DB, dialect db.Dialect, appVersion string) (*File, error) {
	if err := os.MkdirAll(d.Path, 0o700); err != nil {
		return nil, fmt.Errorf("backup directory: %w", err)
	}
	at := time.Now().UTC()
	name := "ragbag-" + at.Format(fileLayout) + ".json.gz"
	tmp, err := os.CreateTemp(d.Path, ".partial-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := Write(ctx, database, dialect, tmp, appVersion); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	path := filepath.Join(d.Path, name)
	if err := os.Rename(tmp.Name(), path); err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &File{Name: name, Size: info.Size(), CreatedAt: at.Truncate(time.Second)}, nil
}

// List returns the backups in the directory, newest first; a missing directory has none.
func (d Dir) List() ([]File, error) {
	entries, err := os.ReadDir(d.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []File{}, nil
	}
	if err != nil {
		return nil, err
	}
	files := []File{}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !fileName.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		stamp := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "ragbag-"), ".json.gz")
		at, err := time.ParseInLocation(fileLayout, stamp, time.UTC)
		if err != nil {
			continue
		}
		files = append(files, File{Name: entry.Name(), Size: info.Size(), CreatedAt: at})
	}
	slices.SortFunc(files, func(a, b File) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return files, nil
}

// Prune deletes all but the newest keep backups and returns how many went.
func (d Dir) Prune(keep int) (int, error) {
	files, err := d.List()
	if err != nil || keep < 1 || len(files) <= keep {
		return 0, err
	}
	removed := 0
	for _, file := range files[keep:] {
		if err := os.Remove(filepath.Join(d.Path, file.Name)); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

// Open returns the path of one of the directory's backups; other names (paths, other files) are refused.
func (d Dir) Open(name string) (string, error) {
	if !fileName.MatchString(name) {
		return "", ErrNoBackup
	}
	path := filepath.Join(d.Path, name)
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return "", ErrNoBackup
	}
	return path, nil
}
