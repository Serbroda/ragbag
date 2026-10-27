// Package store holds all SQL. Queries are portable between SQLite and PostgreSQL and use ? placeholders,
// which are rebound for the driver.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Serbroda/ragbag/internal/db"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ErrNotFound is returned for single-row lookups without a result.
var ErrNotFound = errors.New("not found")

// errRollback ends a preview transaction without reporting an error.
var errRollback = errors.New("rollback")

type Store struct {
	db      *sqlx.DB
	q       sqlx.ExtContext
	inTx    bool
	Dialect db.Dialect
}

func New(database *sqlx.DB, dialect db.Dialect) *Store {
	return &Store{db: database, q: database, Dialect: dialect}
}

// InTx runs fn in a transaction (or the current one) and commits unless fn fails.
func (s *Store) InTx(ctx context.Context, fn func(tx *Store) error) error {
	if s.inTx {
		return fn(s)
	}
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	if err := fn(&Store{db: s.db, q: tx, inTx: true, Dialect: s.Dialect}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// DryRun runs fn in a transaction that is always rolled back: what fn did is reported, nothing is kept.
func (s *Store) DryRun(ctx context.Context, fn func(tx *Store) error) error {
	err := s.InTx(ctx, func(tx *Store) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errRollback
	})
	if errors.Is(err, errRollback) {
		return nil
	}
	return err
}

func (s *Store) get(ctx context.Context, dest any, query string, args ...any) error {
	err := sqlx.GetContext(ctx, s.q, dest, s.q.Rebind(query), args...)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Store) sel(ctx context.Context, dest any, query string, args ...any) error {
	return sqlx.SelectContext(ctx, s.q, dest, s.q.Rebind(query), args...)
}

func (s *Store) exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := s.q.ExecContext(ctx, s.q.Rebind(query), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// selIn expands IN (?) for slice arguments; callers must not pass empty slices.
func (s *Store) selIn(ctx context.Context, dest any, query string, args ...any) error {
	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return err
	}
	return s.sel(ctx, dest, expanded, expandedArgs...)
}

func (s *Store) execIn(ctx context.Context, query string, args ...any) (int64, error) {
	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return 0, err
	}
	return s.exec(ctx, expanded, expandedArgs...)
}

func newID() string {
	return uuid.NewString()
}

// now is truncated to microseconds, the precision PostgreSQL stores, so values compare equal after a round trip.
func now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
