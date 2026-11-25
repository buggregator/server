package explain

import (
	"database/sql"
	"embed"
	"errors"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Caps of the stored conversations.
const (
	maxMessages    = 200 // per thread; older ones drop off
	maxPageThreads = 50  // page threads kept; event threads go with their event
	maxRecent      = 50
)

// Message is one line of a thread: the person's words or an agent's answer.
type Message struct {
	From    string    `json:"from"` // "you" or "agent"
	Role    string    `json:"role,omitempty"`
	Project string    `json:"project,omitempty"`
	Text    string    `json:"text"`
	Context bool      `json:"context,omitempty"` // the context text sent first
	At      time.Time `json:"at"`
}

// Thread is the conversation with the agents about one context.
type Thread struct {
	ID           int64     `json:"id"`
	Kind         string    `json:"kind"`
	Key          string    `json:"key"`
	Title        string    `json:"title"`
	Project      string    `json:"project"`
	Conversation string    `json:"conversation"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Messages     []Message `json:"messages"`
}

// Summary is a thread in the recent list.
type Summary struct {
	ID        int64     `json:"id"`
	Kind      string    `json:"kind"`
	Key       string    `json:"key"`
	Title     string    `json:"title"`
	Project   string    `json:"project"`
	UpdatedAt time.Time `json:"updated_at"`
	LastFrom  string    `json:"last_from"`
}

var errNoThread = errors.New("no such thread")

// threads keeps the conversations in SQLite, so a follow-up after a restart
// still reaches the same h2a conversation.
type threads struct{ db *sql.DB }

const timeFormat = time.RFC3339Nano

func parseTime(s string) time.Time {
	t, _ := time.Parse(timeFormat, s)
	return t
}

const threadCols = `id, kind, key, title, project, conversation, created_at, updated_at`

func scanThread(row interface{ Scan(...any) error }) (Thread, error) {
	var th Thread
	var created, updated string
	err := row.Scan(&th.ID, &th.Kind, &th.Key, &th.Title, &th.Project, &th.Conversation, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Thread{}, errNoThread
	}
	th.CreatedAt, th.UpdatedAt = parseTime(created), parseTime(updated)
	return th, err
}

func (t *threads) withMessages(th Thread, err error) (Thread, error) {
	if err != nil {
		return th, err
	}
	rows, err := t.db.Query(`SELECT sender, role, project, text, context, at FROM assist_messages WHERE thread_id = ? ORDER BY id`, th.ID)
	if err != nil {
		return th, err
	}
	defer rows.Close()
	th.Messages = []Message{}
	for rows.Next() {
		var m Message
		var at string
		if err := rows.Scan(&m.From, &m.Role, &m.Project, &m.Text, &m.Context, &at); err != nil {
			return th, err
		}
		m.At = parseTime(at)
		th.Messages = append(th.Messages, m)
	}
	return th, rows.Err()
}

func (t *threads) byKey(kind, key string) (Thread, error) {
	return t.withMessages(scanThread(t.db.QueryRow(`SELECT `+threadCols+` FROM assist_threads WHERE kind = ? AND key = ?`, kind, key)))
}

func (t *threads) byID(id int64) (Thread, error) {
	return t.withMessages(scanThread(t.db.QueryRow(`SELECT `+threadCols+` FROM assist_threads WHERE id = ?`, id)))
}

func (t *threads) byConversation(conv string) (Thread, error) {
	return scanThread(t.db.QueryRow(`SELECT `+threadCols+` FROM assist_threads WHERE conversation = ? ORDER BY id DESC LIMIT 1`, conv))
}

// start records a new conversation for the context with its first messages,
// replacing an older one of the same context.
func (t *threads) start(c Context, project, conversation string, first []Message) (Thread, error) {
	now := time.Now().UTC().Format(timeFormat)
	tx, err := t.db.Begin()
	if err != nil {
		return Thread{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM assist_messages WHERE thread_id IN (SELECT id FROM assist_threads WHERE kind = ? AND key = ?)`, c.Kind, c.Key); err != nil {
		return Thread{}, err
	}
	if _, err := tx.Exec(`DELETE FROM assist_threads WHERE kind = ? AND key = ?`, c.Kind, c.Key); err != nil {
		return Thread{}, err
	}
	res, err := tx.Exec(`INSERT INTO assist_threads (kind, key, title, project, conversation, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.Kind, c.Key, c.Title, project, conversation, now, now)
	if err != nil {
		return Thread{}, err
	}
	id, _ := res.LastInsertId()
	for _, m := range first {
		if err := insertMessage(tx, id, m); err != nil {
			return Thread{}, err
		}
	}
	if c.Kind == KindPage {
		if err := capPageThreads(tx); err != nil {
			return Thread{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Thread{}, err
	}
	return t.byID(id)
}

// add appends a message to a thread and keeps the last maxMessages.
func (t *threads) add(id int64, m Message) (Thread, error) {
	tx, err := t.db.Begin()
	if err != nil {
		return Thread{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE assist_threads SET updated_at = ? WHERE id = ?`, time.Now().UTC().Format(timeFormat), id)
	if err != nil {
		return Thread{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Thread{}, errNoThread
	}
	if err := insertMessage(tx, id, m); err != nil {
		return Thread{}, err
	}
	if _, err := tx.Exec(`DELETE FROM assist_messages WHERE thread_id = ? AND id NOT IN
		(SELECT id FROM assist_messages WHERE thread_id = ? ORDER BY id DESC LIMIT ?)`, id, id, maxMessages); err != nil {
		return Thread{}, err
	}
	if err := tx.Commit(); err != nil {
		return Thread{}, err
	}
	return t.byID(id)
}

func insertMessage(tx *sql.Tx, id int64, m Message) error {
	if m.At.IsZero() {
		m.At = time.Now().UTC()
	}
	_, err := tx.Exec(`INSERT INTO assist_messages (thread_id, sender, role, project, text, context, at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, m.From, m.Role, m.Project, m.Text, m.Context, m.At.UTC().Format(timeFormat))
	return err
}

func capPageThreads(tx *sql.Tx) error {
	const old = `SELECT id FROM assist_threads WHERE kind = 'page' ORDER BY updated_at DESC, id DESC LIMIT -1 OFFSET ?`
	if _, err := tx.Exec(`DELETE FROM assist_messages WHERE thread_id IN (`+old+`)`, maxPageThreads); err != nil {
		return err
	}
	_, err := tx.Exec(`DELETE FROM assist_threads WHERE id IN (`+old+`)`, maxPageThreads)
	return err
}

// recent lists the last threads, newest first.
func (t *threads) recent(n int) ([]Summary, error) {
	if n <= 0 || n > maxRecent {
		n = maxRecent
	}
	rows, err := t.db.Query(`SELECT t.id, t.kind, t.key, t.title, t.project, t.updated_at,
		COALESCE((SELECT sender FROM assist_messages m WHERE m.thread_id = t.id ORDER BY m.id DESC LIMIT 1), '')
		FROM assist_threads t ORDER BY t.updated_at DESC, t.id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var s Summary
		var updated string
		if err := rows.Scan(&s.ID, &s.Kind, &s.Key, &s.Title, &s.Project, &updated, &s.LastFrom); err != nil {
			return nil, err
		}
		s.UpdatedAt = parseTime(updated)
		out = append(out, s)
	}
	return out, rows.Err()
}

// removeEvents drops the threads of deleted events, in chunks (a clear-all
// names every event).
func (t *threads) removeEvents(uuids []string) error {
	for len(uuids) > 0 {
		n := min(len(uuids), 500)
		if err := t.removeChunk(uuids[:n]); err != nil {
			return err
		}
		uuids = uuids[n:]
	}
	return nil
}

func (t *threads) removeChunk(uuids []string) error {
	args := make([]any, len(uuids))
	for i, u := range uuids {
		args[i] = u
	}
	in := "(" + strings.TrimSuffix(strings.Repeat("?,", len(uuids)), ",") + ")"
	tx, err := t.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM assist_messages WHERE thread_id IN (SELECT id FROM assist_threads WHERE kind = 'event' AND key IN `+in+`)`, args...); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM assist_threads WHERE kind = 'event' AND key IN `+in, args...); err != nil {
		return err
	}
	return tx.Commit()
}
