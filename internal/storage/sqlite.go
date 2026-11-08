package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements event.Store using SQLite.
type SQLiteStore struct {
	db       *sql.DB
	onDelete []func(uuids []string)
}

// Open creates a new SQLite database connection with a single connection.
func Open(dsn string) (*sql.DB, error) {
	return OpenPooled(dsn, 1)
}

// OpenPooled opens the database with a pool of maxOpen connections.
//
// A single connection is only required for writing: in WAL mode readers block
// neither the writer nor each other, while one shared connection turns every
// heavy SELECT into a queue for ingestion — the UI and the incoming events wait
// for each other.
//
// There is one correctness requirement: WAL still allows a single writer, so
// with maxOpen > 1 the DSN needs busy_timeout, otherwise a second write fails
// with SQLITE_BUSY instead of waiting. Both that and _txlock=immediate are
// added when absent.
func OpenPooled(dsn string, maxOpen int) (*sql.DB, error) {
	if dsn == ":memory:" {
		dsn = "file::memory:?cache=shared&_pragma=journal_mode(WAL)"
	}
	if maxOpen < 1 {
		maxOpen = 1
	}
	// An in-memory database lives in a cache shared by all connections, and
	// shared-cache mode uses table locks that busy_timeout does not cover: a
	// pool fails with "database table is locked" (SQLITE_LOCKED_SHAREDCACHE)
	// under concurrent writes. Keep it on one connection.
	if maxOpen > 1 && isInMemoryDSN(dsn) {
		slog.Info("storage: in-memory database, connection pool reduced to 1")
		maxOpen = 1
	}
	if maxOpen > 1 {
		add := func(param, msg string) {
			sep := "?"
			if strings.Contains(dsn, "?") {
				sep = "&"
			}
			if !strings.HasPrefix(dsn, "file:") {
				dsn = "file:" + dsn
			}
			dsn += sep + param
			slog.Info(msg)
		}

		if !strings.Contains(dsn, "busy_timeout") {
			add("_pragma=busy_timeout(10000)",
				"storage: busy_timeout missing from the DSN, added — a connection pool would otherwise get SQLITE_BUSY")
		}

		// BEGIN IMMEDIATE instead of BEGIN DEFERRED for every transaction.
		//
		// Without it busy_timeout does not help writes: a deferred transaction
		// starts as a reader and tries to upgrade the lock on its first write —
		// and if a writer already holds it, SQLite returns SQLITE_BUSY at once
		// without waiting out the timeout (waiting would deadlock two readers
		// that both want to write). Verified on a live stream: a pool of four
		// connections started logging "upsert sentry_traces: database is locked
		// (5)". Every transaction in the code base is a writing one, so taking
		// the write lock upfront costs nothing.
		if !strings.Contains(dsn, "_txlock") {
			add("_txlock=immediate",
				"storage: _txlock missing from the DSN, added immediate — a pooled write would otherwise get SQLITE_BUSY without waiting")
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxOpen)
	return db, nil
}

// isInMemoryDSN reports whether dsn names an in-memory database:
// ":memory:", "file::memory:…" or a URI with mode=memory.
func isInMemoryDSN(dsn string) bool {
	return strings.Contains(dsn, ":memory:") || strings.Contains(dsn, "mode=memory")
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// OnDelete registers fn to run after Delete or DeleteAll removed events.
// fn gets the UUIDs of the removed rows, so files kept outside SQLite
// (attachments, trace files) can be removed with them. Register hooks at
// startup, before the store is used.
func (s *SQLiteStore) OnDelete(fn func(uuids []string)) {
	s.onDelete = append(s.onDelete, fn)
}

// deleteReturning runs a DELETE ... RETURNING uuid and passes the removed
// UUIDs to the OnDelete hooks.
func (s *SQLiteStore) deleteReturning(ctx context.Context, query string, args ...any) error {
	rows, err := s.db.QueryContext(ctx, query+" RETURNING uuid", args...)
	if err != nil {
		return err
	}
	var uuids []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			rows.Close()
			return err
		}
		uuids = append(uuids, u)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(uuids) == 0 {
		return nil
	}
	for _, fn := range s.onDelete {
		fn(uuids)
	}
	return nil
}

func (s *SQLiteStore) Store(ctx context.Context, ev event.Event) error {
	payload, _ := json.Marshal(json.RawMessage(ev.Payload))
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO events (uuid, type, payload, timestamp, project, is_pinned) VALUES (?, ?, ?, ?, ?, ?)`,
		ev.UUID, ev.Type, string(payload), fmt.Sprintf("%.6f", ev.Timestamp), ev.Project, boolToInt(ev.IsPinned),
	)
	return err
}

func (s *SQLiteStore) FindByUUID(ctx context.Context, uuid string) (*event.Event, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT uuid, type, payload, timestamp, project, is_pinned FROM events WHERE uuid = ?`, uuid,
	)
	return scanEvent(row)
}

func (s *SQLiteStore) FindAll(ctx context.Context, opts event.FindOptions) ([]event.Event, error) {
	query := `SELECT uuid, type, payload, timestamp, project, is_pinned FROM events`
	var args []any
	var conditions []string

	if opts.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, opts.Type)
	}
	if opts.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, opts.Project)
	}
	// Window bounds are compared numerically: timestamp is stored as text
	// (fmt.Sprintf("%.6f")), and a string comparison would break as soon as the
	// number of digits in unix seconds changes.
	if opts.From > 0 {
		conditions = append(conditions, "CAST(timestamp AS REAL) >= ?")
		args = append(args, opts.From)
	}
	if opts.To > 0 {
		conditions = append(conditions, "CAST(timestamp AS REAL) <= ?")
		args = append(args, opts.To)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if opts.PinnedFirst {
		query += " ORDER BY is_pinned DESC, timestamp DESC"
	} else {
		query += " ORDER BY timestamp DESC"
	}

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	} else if opts.Offset > 0 {
		query += " LIMIT -1" // SQLite needs a LIMIT before OFFSET; -1 = no limit
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []event.Event
	for rows.Next() {
		ev, err := scanEventRows(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *ev)
	}
	return events, rows.Err()
}

func (s *SQLiteStore) Delete(ctx context.Context, uuid string) error {
	return s.deleteReturning(ctx, `DELETE FROM events WHERE uuid = ?`, uuid)
}

func (s *SQLiteStore) DeleteAll(ctx context.Context, opts event.DeleteOptions) error {
	keep := ""
	if opts.KeepPinned {
		keep = " AND is_pinned = 0"
	}
	if len(opts.UUIDs) > 0 {
		placeholders := make([]string, len(opts.UUIDs))
		args := make([]any, len(opts.UUIDs))
		for i, u := range opts.UUIDs {
			placeholders[i] = "?"
			args[i] = u
		}
		return s.deleteReturning(ctx,
			`DELETE FROM events WHERE uuid IN (`+strings.Join(placeholders, ",")+`)`+keep, args...)
	}

	query := `DELETE FROM events`
	var args []any
	var conditions []string

	if opts.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, opts.Type)
	}
	if opts.Project != "" {
		conditions = append(conditions, "project = ?")
		args = append(args, opts.Project)
	}
	if opts.KeepPinned {
		conditions = append(conditions, "is_pinned = 0")
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	return s.deleteReturning(ctx, query, args...)
}

func (s *SQLiteStore) UpdatePayload(ctx context.Context, uuid string, payload json.RawMessage) error {
	res, err := s.db.ExecContext(ctx, `UPDATE events SET payload = ? WHERE uuid = ?`, string(payload), uuid)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return event.ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) Pin(ctx context.Context, uuid string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE events SET is_pinned = 1 WHERE uuid = ?`, uuid)
	return err
}

func (s *SQLiteStore) Unpin(ctx context.Context, uuid string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE events SET is_pinned = 0 WHERE uuid = ?`, uuid)
	return err
}

type scannable interface {
	Scan(dest ...any) error
}

func scanEvent(row scannable) (*event.Event, error) {
	var ev event.Event
	var payload string
	var ts string
	var pinned int

	if err := row.Scan(&ev.UUID, &ev.Type, &payload, &ts, &ev.Project, &pinned); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	ev.Payload = json.RawMessage(payload)
	fmt.Sscanf(ts, "%f", &ev.Timestamp)
	ev.IsPinned = pinned != 0
	return &ev, nil
}

func scanEventRows(rows *sql.Rows) (*event.Event, error) {
	var ev event.Event
	var payload string
	var ts string
	var pinned int

	if err := rows.Scan(&ev.UUID, &ev.Type, &payload, &ts, &ev.Project, &pinned); err != nil {
		return nil, err
	}

	ev.Payload = json.RawMessage(payload)
	fmt.Sscanf(ts, "%f", &ev.Timestamp)
	ev.IsPinned = pinned != 0
	return &ev, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
