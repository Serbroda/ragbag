package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestOpenMigratesAndIsIdempotent(t *testing.T) {
	url := "sqlite:" + t.TempDir() + "/nested/dir/ragbag.db"
	for range 2 {
		database, dialect, err := Open(context.Background(), url)
		require.NoError(t, err)
		require.Equal(t, SQLite, dialect)
		var tables int
		require.NoError(t, database.Get(&tables, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'users'"))
		require.Equal(t, 1, tables)
		database.Close()
	}
}

func TestDatabasesOfTheJavaVersionAreRefused(t *testing.T) {
	path := t.TempDir() + "/old.db"
	old, err := sqlx.Open("sqlite", "file:"+path)
	require.NoError(t, err)
	_, err = old.Exec("CREATE TABLE databasechangelog (id TEXT)")
	require.NoError(t, err)
	old.Close()

	_, _, err = Open(context.Background(), "sqlite:"+path)
	require.True(t, errors.Is(err, ErrJavaDatabase), "got %v", err)
}

func TestUnknownDatabaseURLs(t *testing.T) {
	_, _, err := Open(context.Background(), "jdbc:h2:mem:test")
	require.Error(t, err)
}
