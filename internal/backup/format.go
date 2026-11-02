package backup

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// The file: {"format":…,"format_version":…,"schema_version":…,"app_version":…,"created_at":…,
// "tables":[{"name":"users","columns":["id",…],"rows":[["…",…],…]},…]}, read as a stream (images make it big).

// reader goes through a backup table by table and row by row.
type reader struct {
	file   *os.File
	zipped *gzip.Reader
	dec    *json.Decoder
	header Header
}

var errInvalid = errors.New("this is no valid Ragbag backup")

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errInvalid, fmt.Sprintf(format, args...))
}

func readHeader(path string) (*Header, error) {
	r, err := open(path)
	if err != nil {
		return nil, err
	}
	defer r.close()
	return &r.header, nil
}

// open reads the header (up to the tables) and checks it.
func open(path string) (*reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := &reader{file: file}
	buffered := bufio.NewReader(file)
	var input io.Reader = buffered
	// gzip, or plain JSON when somebody unpacked it
	if magic, _ := buffered.Peek(2); len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		if r.zipped, err = gzip.NewReader(buffered); err != nil {
			r.close()
			return nil, invalid("%v", err)
		}
		input = r.zipped
	}
	r.dec = json.NewDecoder(input)
	r.dec.UseNumber()
	if err := r.readHeader(); err != nil {
		r.close()
		return nil, err
	}
	return r, nil
}

func (r *reader) close() {
	if r.zipped != nil {
		r.zipped.Close()
	}
	r.file.Close()
}

func (r *reader) expect(delim json.Delim) error {
	token, err := r.dec.Token()
	if err != nil {
		return invalid("%v", err)
	}
	if token != delim {
		return invalid("expected %v, found %v", delim, token)
	}
	return nil
}

func (r *reader) key() (string, error) {
	token, err := r.dec.Token()
	if err != nil {
		return "", invalid("%v", err)
	}
	key, ok := token.(string)
	if !ok {
		return "", invalid("expected a key, found %v", token)
	}
	return key, nil
}

func (r *reader) skip() error {
	var ignored json.RawMessage
	if err := r.dec.Decode(&ignored); err != nil {
		return invalid("%v", err)
	}
	return nil
}

func (r *reader) readHeader() error {
	if err := r.expect('{'); err != nil {
		return err
	}
	fields := map[string]any{
		"format": &r.header.Format, "format_version": &r.header.FormatVersion,
		"schema_version": &r.header.SchemaVersion, "app_version": &r.header.AppVersion,
		"created_at": &r.header.CreatedAt,
	}
	for r.dec.More() {
		key, err := r.key()
		if err != nil {
			return err
		}
		if key == "tables" {
			if r.header.Format != Format {
				return invalid("exports of a single collection are imported in that collection's menu")
			}
			if r.header.FormatVersion < 1 || r.header.FormatVersion > FormatVersion {
				return fmt.Errorf("this backup has format version %d, which this version of Ragbag does not know: "+
					"update Ragbag first", r.header.FormatVersion)
			}
			if r.header.SchemaVersion < 1 {
				return invalid("schema_version is missing")
			}
			return r.expect('[')
		}
		field, ok := fields[key]
		if !ok {
			if err := r.skip(); err != nil {
				return err
			}
			continue
		}
		if err := r.dec.Decode(field); err != nil {
			return invalid("%s: %v", key, err)
		}
	}
	if r.header.Format != Format {
		return invalid("exports of a single collection are imported in that collection's menu")
	}
	return invalid("no tables")
}

// nextTable reads up to the rows of the next table; io.EOF after the last one.
func (r *reader) nextTable() (name string, columns []string, err error) {
	if !r.dec.More() {
		if err := r.expect(']'); err != nil {
			return "", nil, err
		}
		return "", nil, io.EOF
	}
	if err := r.expect('{'); err != nil {
		return "", nil, err
	}
	for r.dec.More() {
		key, err := r.key()
		if err != nil {
			return "", nil, err
		}
		switch key {
		case "name":
			err = r.dec.Decode(&name)
		case "columns":
			err = r.dec.Decode(&columns)
		case "rows":
			if name == "" || len(columns) == 0 {
				return "", nil, invalid("a table without name or columns")
			}
			return name, columns, r.expect('[')
		default:
			err = r.skip()
		}
		if err != nil {
			return "", nil, invalid("%v", err)
		}
	}
	return "", nil, invalid("table %q has no rows", name)
}

// eachRow hands over the rows of the current table and reads up to the next one.
func (r *reader) eachRow(fn func([]any) error) error {
	for r.dec.More() {
		var row []any
		if err := r.dec.Decode(&row); err != nil {
			return invalid("%v", err)
		}
		if err := fn(row); err != nil {
			return err
		}
	}
	if err := r.expect(']'); err != nil {
		return err
	}
	for r.dec.More() {
		if _, err := r.key(); err != nil {
			return err
		}
		if err := r.skip(); err != nil {
			return err
		}
	}
	return r.expect('}')
}

// --- values: the same in the file whatever the database ---

type kind int

const (
	kindText kind = iota
	kindInt
	kindBool
	kindTime
	kindBytes
)

// kindOf maps the column types of SQLite (as declared) and PostgreSQL (pgx names).
func kindOf(databaseType string) kind {
	switch t := strings.ToUpper(databaseType); {
	case strings.Contains(t, "TIME"), t == "DATE":
		return kindTime
	case t == "BLOB", t == "BYTEA":
		return kindBytes
	case strings.HasPrefix(t, "BOOL"):
		return kindBool
	case strings.Contains(t, "INT"):
		return kindInt
	default:
		return kindText
	}
}

func targetColumns(ctx context.Context, tx *sqlx.Tx, name string) (map[string]kind, error) {
	rows, err := tx.QueryContext(ctx, "SELECT * FROM "+name+" WHERE 1 = 0")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	kinds := make(map[string]kind, len(types))
	for _, column := range types {
		kinds[column.Name()] = kindOf(column.DatabaseTypeName())
	}
	return kinds, nil
}

// Times as RFC 3339 in UTC, binary data as base64; booleans stay as the database returns them (SQLite: 0/1),
// decode converts them for the target.
func encode(value any, k kind) (any, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano), nil
	case []byte:
		if k == kindBytes {
			return base64.StdEncoding.EncodeToString(v), nil
		}
		return encode(string(v), k)
	case string:
		if k == kindTime {
			t, err := parseTime(v)
			if err != nil {
				return nil, err
			}
			return t.Format(time.RFC3339Nano), nil
		}
		return v, nil
	default:
		return v, nil
	}
}

func decode(value any, k kind) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch k {
	case kindInt:
		switch v := value.(type) {
		case json.Number:
			return v.Int64()
		case bool:
			if v {
				return int64(1), nil
			}
			return int64(0), nil
		}
	case kindBool:
		switch v := value.(type) {
		case bool:
			return v, nil
		case json.Number:
			return v.String() != "0", nil
		}
	case kindTime:
		if v, ok := value.(string); ok {
			return parseTime(v)
		}
	case kindBytes:
		if v, ok := value.(string); ok {
			return base64.StdEncoding.DecodeString(v)
		}
	default:
		switch v := value.(type) {
		case string:
			return v, nil
		case json.Number:
			return v.String(), nil
		}
	}
	return nil, fmt.Errorf("unexpected value %v", value)
}

var timeLayouts = []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999"}

// parseTime: stored times are UTC, truncated to microseconds like everything the store writes.
func parseTime(value string) (time.Time, error) {
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC().Truncate(time.Microsecond), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time %q", value)
}
