// Package backup writes the whole instance (accounts, spaces, links, images, everything) into one gzip
// compressed JSON file and restores it, independent of the database: SQLite to PostgreSQL and back.
//
// Rows keep their ids, so share links, bookmarklets and password hashes stay valid. Backups of older versions
// are restored into a temporary SQLite database at their schema version and migrated from there, so data
// migrations run on them as on any database.
package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/db"
	"github.com/jmoiron/sqlx"
)

const (
	Format        = "ragbag-instance"
	FormatVersion = 1
)

// table is a table in a backup. They are listed in an order that satisfies the foreign keys.
type table struct {
	name string
	// deferred columns reference tables restored later; they are set at the end
	deferred []string
	// parent references the same table: rows are inserted parents first
	parent string
	// since is the migration that added the table, when it is newer than the oldest backups (version 10)
	since int64
}

var tables = []table{
	{name: "users", deferred: []string{"start_space_id"}},
	{name: "images"},
	{name: "spaces"},
	{name: "space_members"},
	{name: "collections", parent: "parent_id"},
	{name: "collection_members"},
	{name: "bookmarks"},
	{name: "bookmark_tags"},
	{name: "bookmark_favorites"},
	{name: "bookmark_reactions"},
	{name: "dashboard_sections"},
	{name: "dashboard_items"},
	{name: "invites"},
	{name: "collection_shares"},
	{name: "collection_visits"},
	{name: "collection_locks"},
	{name: "space_bars"},
	{name: "abuse_reports", since: 13},
	{name: "api_tokens", since: 22},
}

// Skipped holds what a backup leaves out on purpose: pending reset and confirmation links, which are only
// mailed or handed to one person, and the migration bookkeeping.
var Skipped = []string{"password_resets", "email_verifications", "goose_db_version"}

// Tables lists the tables a backup holds.
func Tables() []string {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = t.name
	}
	return names
}

type Header struct {
	Format        string    `json:"format"`
	FormatVersion int       `json:"format_version"`
	SchemaVersion int64     `json:"schema_version"`
	AppVersion    string    `json:"app_version"`
	CreatedAt     time.Time `json:"created_at"`
}

// Summary describes a backup: its header and the number of rows per table.
type Summary struct {
	Header
	Rows map[string]int `json:"rows"`
	// Accounts are the restored accounts (without deleted ones), to tell who will be an admin
	Accounts []Account `json:"accounts"`
}

type Account struct {
	Username   string `db:"username" json:"username"`
	SetupAdmin bool   `db:"setup_admin" json:"setup_admin"`
}

// ErrNotEmpty: a backup only replaces an instance without data (nothing but the setup admin).
var ErrNotEmpty = errors.New("this instance already has data: a backup can only be restored into an empty one")

// Write streams a backup of the database to w.
func Write(ctx context.Context, database *sqlx.DB, dialect db.Dialect, w io.Writer, appVersion string) error {
	version, err := db.SchemaVersion(ctx, database, dialect)
	if err != nil {
		return err
	}
	return write(ctx, database, w, Header{Format: Format, FormatVersion: FormatVersion, SchemaVersion: version,
		AppVersion: appVersion, CreatedAt: time.Now().UTC()})
}

func write(ctx context.Context, database *sqlx.DB, w io.Writer, header Header) error {
	// One transaction: every table from the same moment
	options := &sql.TxOptions{}
	if database.DriverName() != "sqlite" {
		options = &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	}
	tx, err := database.BeginTxx(ctx, options)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	zipped := gzip.NewWriter(w)
	out := bufio.NewWriter(zipped)
	head, err := json.Marshal(header)
	if err != nil {
		return err
	}
	// The header first, so reading a backup can check it before going through the rows
	out.Write(head[:len(head)-1])
	out.WriteString(`,"tables":[`)
	first := true
	for _, t := range tables {
		// Databases at an older schema (only a temporary one while upgrading a backup)
		if t.since > header.SchemaVersion {
			continue
		}
		if !first {
			out.WriteString(",")
		}
		first = false
		if err := writeTable(ctx, tx, out, t.name); err != nil {
			return fmt.Errorf("backup of %s: %w", t.name, err)
		}
	}
	out.WriteString("]}\n")
	if err := out.Flush(); err != nil {
		return err
	}
	return zipped.Close()
}

func writeTable(ctx context.Context, tx *sqlx.Tx, out *bufio.Writer, name string) error {
	rows, err := tx.QueryContext(ctx, "SELECT * FROM "+name)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns, err := rows.ColumnTypes()
	if err != nil {
		return err
	}
	names := make([]string, len(columns))
	kinds := make([]kind, len(columns))
	for i, column := range columns {
		names[i], kinds[i] = column.Name(), kindOf(column.DatabaseTypeName())
	}
	meta, _ := json.Marshal(struct {
		Name    string   `json:"name"`
		Columns []string `json:"columns"`
	}{name, names})
	out.Write(meta[:len(meta)-1])
	out.WriteString(`,"rows":[`)
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	for first := true; rows.Next(); first = false {
		if err := rows.Scan(pointers...); err != nil {
			return err
		}
		row := make([]any, len(values))
		for i, value := range values {
			if row[i], err = encode(value, kinds[i]); err != nil {
				return fmt.Errorf("column %s: %w", names[i], err)
			}
		}
		line, err := json.Marshal(row)
		if err != nil {
			return err
		}
		if !first {
			out.WriteString(",")
		}
		out.WriteString("\n")
		out.Write(line)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	out.WriteString("]}")
	return nil
}

// Restore replaces the data of an empty instance with the backup at path; dryRun only checks it (every row is
// inserted and rolled back). Older backups are migrated on the way.
func Restore(ctx context.Context, database *sqlx.DB, dialect db.Dialect, path string, dryRun bool) (*Summary, error) {
	header, err := readHeader(path)
	if err != nil {
		return nil, err
	}
	current, err := db.SchemaVersion(ctx, database, dialect)
	if err != nil {
		return nil, err
	}
	if header.SchemaVersion > current {
		return nil, fmt.Errorf("this backup was made by a newer version of Ragbag (%s, schema %d, this one has %d): "+
			"update Ragbag first", header.AppVersion, header.SchemaVersion, current)
	}
	if header.SchemaVersion < current {
		upgraded, cleanup, err := upgrade(ctx, path, header)
		if err != nil {
			return nil, fmt.Errorf("migrating the backup: %w", err)
		}
		defer cleanup()
		path = upgraded
	}
	summary, err := restore(ctx, database, path, dryRun, true)
	if err != nil {
		return nil, err
	}
	// The original's, not the one of the upgraded copy
	summary.Header = *header
	return summary, nil
}

// upgrade restores the backup into a temporary SQLite database at its schema version, migrates that and backs
// it up again.
func upgrade(ctx context.Context, path string, header *Header) (string, func(), error) {
	dir, err := os.MkdirTemp("", "ragbag-restore-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	database, err := db.OpenSQLiteAt(ctx, filepath.Join(dir, "upgrade.db"), header.SchemaVersion)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	defer database.Close()
	if _, err := restore(ctx, database, path, false, false); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := db.MigrateTo(ctx, database, db.SQLite, 0); err != nil {
		cleanup()
		return "", nil, err
	}
	upgraded := filepath.Join(dir, "upgraded.json.gz")
	file, err := os.Create(upgraded)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	defer file.Close()
	version, err := db.SchemaVersion(ctx, database, db.SQLite)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	next := *header
	next.SchemaVersion = version
	if err := write(ctx, database, file, next); err != nil {
		cleanup()
		return "", nil, err
	}
	return upgraded, cleanup, nil
}

// restore loads the backup into database; requireEmpty for the instance (otherwise a fresh temporary database).
func restore(ctx context.Context, database *sqlx.DB, path string, dryRun, requireEmpty bool) (*Summary, error) {
	reader, err := open(path)
	if err != nil {
		return nil, err
	}
	defer reader.close()
	tx, err := database.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if requireEmpty {
		var data int
		err := tx.GetContext(ctx, &data, tx.Rebind(`SELECT (SELECT COUNT(*) FROM spaces) +
			(SELECT COUNT(*) FROM users WHERE setup_admin = ?)`), false)
		if err != nil {
			return nil, err
		}
		if data > 0 {
			return nil, ErrNotEmpty
		}
		// Whatever there is (the setup admin) goes; dependents first
		for i := len(tables) - 1; i >= 0; i-- {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+tables[i].name); err != nil {
				return nil, err
			}
		}
		for _, name := range Skipped[:2] {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+name); err != nil {
				return nil, err
			}
		}
	}
	summary := &Summary{Header: reader.header, Rows: map[string]int{}}
	var updates []deferredUpdate
	for {
		name, columns, err := reader.nextTable()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		loaded, err := loadTable(ctx, tx, reader, name, columns)
		if err != nil {
			return nil, fmt.Errorf("restoring %s: %w", name, err)
		}
		summary.Rows[name] = loaded.rows
		updates = append(updates, loaded.updates...)
	}
	for _, update := range updates {
		if _, err := tx.ExecContext(ctx, tx.Rebind(update.query), update.args...); err != nil {
			return nil, err
		}
	}
	// Only the instance has the current schema; the temporary database of an upgrade may not
	if requireEmpty {
		if err := tx.SelectContext(ctx, &summary.Accounts, tx.Rebind(`SELECT username, setup_admin FROM users
			WHERE deleted_at IS NULL ORDER BY username`)); err != nil {
			return nil, err
		}
	}
	if dryRun {
		return summary, nil
	}
	return summary, tx.Commit()
}

type deferredUpdate struct {
	query string
	args  []any
}

type loaded struct {
	rows    int
	updates []deferredUpdate
}

func loadTable(ctx context.Context, tx *sqlx.Tx, reader *reader, name string, columns []string) (*loaded, error) {
	index := slices.IndexFunc(tables, func(t table) bool { return t.name == name })
	if index < 0 {
		if slices.Contains(Skipped, name) {
			return &loaded{}, reader.eachRow(func([]any) error { return nil })
		}
		return nil, fmt.Errorf("unknown table")
	}
	spec := tables[index]
	kinds, err := targetColumns(ctx, tx, name)
	if err != nil {
		return nil, err
	}
	columnKinds := make([]kind, len(columns))
	for i, column := range columns {
		k, ok := kinds[column]
		if !ok {
			return nil, fmt.Errorf("unknown column %s", column)
		}
		columnKinds[i] = k
	}
	insert, err := tx.PreparexContext(ctx, tx.Rebind(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", name,
		strings.Join(columns, ", "), strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", "))))
	if err != nil {
		return nil, err
	}
	defer insert.Close()

	idColumn := slices.Index(columns, "id")
	var deferred []int
	for _, column := range spec.deferred {
		if i := slices.Index(columns, column); i >= 0 {
			deferred = append(deferred, i)
		}
	}
	parent := -1
	if spec.parent != "" {
		parent = slices.Index(columns, spec.parent)
	}
	if (len(deferred) > 0 || parent >= 0) && idColumn < 0 {
		return nil, errors.New("rows without id")
	}

	result := &loaded{}
	insertRow := func(row []any) error {
		if _, err := insert.ExecContext(ctx, row...); err != nil {
			return err
		}
		result.rows++
		return nil
	}
	// Rows of a table referencing itself wait until the end, then go in parents first
	var pending [][]any
	err = reader.eachRow(func(values []any) error {
		if len(values) != len(columns) {
			return fmt.Errorf("a row has %d values for %d columns", len(values), len(columns))
		}
		row := make([]any, len(values))
		for i, value := range values {
			converted, err := decode(value, columnKinds[i])
			if err != nil {
				return fmt.Errorf("column %s: %w", columns[i], err)
			}
			row[i] = converted
		}
		for _, i := range deferred {
			if row[i] != nil {
				result.updates = append(result.updates, deferredUpdate{
					query: fmt.Sprintf("UPDATE %s SET %s = ? WHERE id = ?", name, columns[i]),
					args:  []any{row[i], row[idColumn]},
				})
				row[i] = nil
			}
		}
		if parent >= 0 {
			pending = append(pending, row)
			return nil
		}
		return insertRow(row)
	})
	if err != nil {
		return nil, err
	}
	done := map[any]bool{}
	for len(pending) > 0 {
		var later [][]any
		for _, row := range pending {
			if row[parent] != nil && !done[row[parent]] {
				later = append(later, row)
				continue
			}
			if err := insertRow(row); err != nil {
				return nil, err
			}
			done[row[idColumn]] = true
		}
		if len(later) == len(pending) {
			return nil, errors.New("rows reference parents that are missing")
		}
		pending = later
	}
	return result, nil
}

// Admins are the restored accounts that will be instance admins: those in adminUsers (ADMIN_USERS), or while
// that is empty the backup's setup admin.
func (s *Summary) Admins(adminUsers []string) []string {
	admins := []string{}
	for _, account := range s.Accounts {
		if len(adminUsers) == 0 && account.SetupAdmin ||
			slices.ContainsFunc(adminUsers, func(name string) bool { return strings.EqualFold(name, account.Username) }) {
			admins = append(admins, account.Username)
		}
	}
	return admins
}
