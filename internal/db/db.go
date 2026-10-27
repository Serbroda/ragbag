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
	dir, err := fs.Sub(migrations, "migrations/"+string(dialect))
	if err != nil {
		return err
	}
	gooseDialect := goose.DialectSQLite3
	if dialect == Postgres {
		gooseDialect = goose.DialectPostgres
	}
	provider, err := goose.NewProvider(gooseDialect, database.DB, dir)
	if err != nil {
		return fmt.Errorf("prepare migrations: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	for _, result := range results {
		slog.Info("applied migration", "version", result.Source.Version, "file", result.Source.Path)
	}
	return nil
}
