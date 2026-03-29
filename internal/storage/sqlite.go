package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements event.Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// Open creates a new SQLite database connection.
func Open(dsn string) (*sql.DB, error) {
	if dsn == ":memory:" {
		dsn = "file::memory:?cache=shared&_pragma=journal_mode(WAL)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite single-writer
	return db, nil
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
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
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
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
	_, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE uuid = ?`, uuid)
	return err
}

func (s *SQLiteStore) DeleteAll(ctx context.Context, opts event.DeleteOptions) error {
	if len(opts.UUIDs) > 0 {
		placeholders := make([]string, len(opts.UUIDs))
		args := make([]any, len(opts.UUIDs))
		for i, u := range opts.UUIDs {
			placeholders[i] = "?"
			args[i] = u
		}
		_, err := s.db.ExecContext(ctx,
			`DELETE FROM events WHERE uuid IN (`+strings.Join(placeholders, ",")+`)`, args...)
		return err
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
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
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
