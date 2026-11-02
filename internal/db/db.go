// Package db opens SQLite or PostgreSQL databases and migrates them.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type Dialect string

const (
	SQLite   Dialect = "sqlite"
	Postgres Dialect = "postgres"
)

//go:embed migrations
var migrations embed.FS

// Open connects to "sqlite:<path>" or a "postgres://" URL and applies pending migrations.
func Open(ctx context.Context, url string) (*sqlx.DB, Dialect, error) {
	var (
		database *sqlx.DB
		dialect  Dialect
		err      error
	)
	switch {
	case strings.HasPrefix(url, "sqlite:"):
		dialect = SQLite
		database, err = openSQLite(strings.TrimPrefix(url, "sqlite:"))
	case strings.HasPrefix(url, "postgres://"), strings.HasPrefix(url, "postgresql://"):
		dialect = Postgres
		database, err = sqlx.Open("pgx", url)
	default:
		return nil, "", fmt.Errorf("DATABASE_URL must start with sqlite: or postgres://, got %q", url)
	}
	if err != nil {
		return nil, "", err
	}
	if err := database.PingContext(ctx); err != nil {
		database.Close()
		return nil, "", fmt.Errorf("connect to database: %w", err)
	}
	if err := migrate(ctx, database, dialect); err != nil {
		database.Close()
		return nil, "", err
	}
	return database, dialect, nil
}

func openSQLite(path string) (*sqlx.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
	}
	// foreign_keys: cascades; WAL: readers alongside the writer; busy_timeout: wait for locks instead of failing;
	// _txlock=immediate: take the write lock at BEGIN so transactions never deadlock on lock upgrades
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)" +
		"&_pragma=synchronous(NORMAL)&_txlock=immediate&_time_format=sqlite"
	database, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(4)
	database.SetConnMaxIdleTime(5 * time.Minute)
	return database, nil
}

// ErrJavaDatabase means the database was created by the former Java (Spring Boot) version of Ragbag.
var ErrJavaDatabase = errors.New("this database was created by the Java version of Ragbag, which uses a different " +
	"schema: export your collections there (JSON export), start this version with a new, empty database " +
	"(new volume, file or PostgreSQL database) and import them")

func migrate(ctx context.Context, database *sqlx.DB, dialect Dialect) error {
	// Liquibase, which the Java version used, keeps its bookkeeping in databasechangelog
	var liquibaseRows int
	if err := database.GetContext(ctx, &liquibaseRows, "SELECT COUNT(*) FROM databasechangelog"); err == nil {
		return ErrJavaDatabase
	}
	return MigrateTo(ctx, database, dialect, 0)
}

func provider(database *sqlx.DB, dialect Dialect) (*goose.Provider, error) {
	dir, err := fs.Sub(migrations, "migrations/"+string(dialect))
	if err != nil {
		return nil, err
	}
	gooseDialect := goose.DialectSQLite3
	if dialect == Postgres {
		gooseDialect = goose.DialectPostgres
	}
	provider, err := goose.NewProvider(gooseDialect, database.DB, dir)
	if err != nil {
		return nil, fmt.Errorf("prepare migrations: %w", err)
	}
	return provider, nil
}

// MigrateTo applies the pending migrations up to version, 0 meaning all of them.
func MigrateTo(ctx context.Context, database *sqlx.DB, dialect Dialect, version int64) error {
	provider, err := provider(database, dialect)
	if err != nil {
		return err
	}
	var results []*goose.MigrationResult
	if version == 0 {
		results, err = provider.Up(ctx)
	} else {
		results, err = provider.UpTo(ctx, version)
	}
	if err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	for _, result := range results {
		slog.Info("applied migration", "version", result.Source.Version, "file", result.Source.Path)
	}
	return nil
}

// SchemaVersion is the version of the last applied migration.
func SchemaVersion(ctx context.Context, database *sqlx.DB, dialect Dialect) (int64, error) {
	provider, err := provider(database, dialect)
	if err != nil {
		return 0, err
	}
	return provider.GetDBVersion(ctx)
}

// LatestVersion is the version of the newest migration this build knows.
func LatestVersion(dialect Dialect) int64 {
	entries, _ := fs.ReadDir(migrations, "migrations/"+string(dialect))
	var latest int64
	for _, entry := range entries {
		number, _, _ := strings.Cut(entry.Name(), "_")
		if version, err := strconv.ParseInt(number, 10, 64); err == nil && version > latest {
			latest = version
		}
	}
	return latest
}

// OpenSQLiteAt opens a database migrated only up to version (upgrading backups of older versions).
func OpenSQLiteAt(ctx context.Context, path string, version int64) (*sqlx.DB, error) {
	database, err := openSQLite(path)
	if err != nil {
		return nil, err
	}
	if err := MigrateTo(ctx, database, SQLite, version); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

// Handle is an open database with its dialect.
type Handle struct {
	DB      *sqlx.DB
	Dialect Dialect
	URL     string
}
