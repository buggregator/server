package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// timeFormat keeps stored times fixed-width, so they sort as strings.
const timeFormat = "2006-01-02T15:04:05.000000Z"

func formatTime(t time.Time) string { return t.UTC().Format(timeFormat) }

// UserStore keeps signed-in users, their sessions, their settings and the
// audit log in the server's database (tables from migration 000044).
type UserStore struct {
	db  *sql.DB
	now func() time.Time
}

// NewUserStore returns a store over db; the tables must exist.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db, now: time.Now}
}

// Upsert saves the provider's user (matched by provider and subject) and
// returns its id. Profile fields follow the provider on every sign-in.
func (s *UserStore) Upsert(u User) (string, error) {
	subject := u.Subject
	if subject == "" {
		subject = "email:" + u.Email
	}
	now := formatTime(s.now())
	var id string
	err := s.db.QueryRow(`INSERT INTO users (id, provider, subject, email, name, avatar, hosted_domain, created_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, subject) DO UPDATE SET
			email = excluded.email, name = excluded.name, avatar = excluded.avatar,
			hosted_domain = excluded.hosted_domain, last_login_at = excluded.last_login_at
		RETURNING id`,
		randomID(), u.Provider, subject, u.Email, u.Username, u.Avatar, u.HostedDomain, now, now).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert user: %w", err)
	}
	return id, nil
}

// CreateSession opens a session for the user until exp and returns its id.
func (s *UserStore) CreateSession(userID, userAgent, ip string, exp time.Time) (string, error) {
	id := randomID()
	_, err := s.db.Exec(`INSERT INTO sessions (id, user_id, created_at, expires_at, user_agent, ip) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, formatTime(s.now()), formatTime(exp), truncate(userAgent, 512), ip)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

// Revoke ends a session; its token stops working at once.
func (s *UserStore) Revoke(sessionID string) error {
	_, err := s.db.Exec(`UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, formatTime(s.now()), sessionID)
	return err
}

// SessionValid reports whether the session exists, is not revoked and has
// not expired.
func (s *UserStore) SessionValid(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ? AND revoked_at IS NULL AND expires_at > ?`,
		sessionID, formatTime(s.now())).Scan(&n)
	return err == nil && n == 1
}

// Settings returns the user's stored settings, key → JSON value.
func (s *UserStore) Settings(userID string) (map[string]json.RawMessage, error) {
	rows, err := s.db.Query(`SELECT key, value FROM user_settings WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = json.RawMessage(v)
	}
	return out, rows.Err()
}

// ErrTooManySettings: the change would leave the user more keys than allowed.
var ErrTooManySettings = errors.New("too many settings")

// PutSettings merges values into the user's settings; a JSON null removes
// the key. With maxKeys > 0 a change that would leave more keys is refused
// (ErrTooManySettings), checked in the same transaction.
func (s *UserStore) PutSettings(userID string, values map[string]json.RawMessage, maxKeys int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := formatTime(s.now())
	for k, v := range values {
		if string(v) == "null" {
			_, err = tx.Exec(`DELETE FROM user_settings WHERE user_id = ? AND key = ?`, userID, k)
		} else {
			_, err = tx.Exec(`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES (?, ?, ?, ?)
				ON CONFLICT(user_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
				userID, k, string(v), now)
		}
		if err != nil {
			return err
		}
	}
	if maxKeys > 0 {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM user_settings WHERE user_id = ?`, userID).Scan(&n); err != nil {
			return err
		}
		if n > maxKeys {
			return ErrTooManySettings
		}
	}
	return tx.Commit()
}

// AuditEntry is one row of the audit log.
type AuditEntry struct {
	ID        int64  `json:"id"`
	At        string `json:"at"`
	UserID    string `json:"user_id"`
	UserLabel string `json:"user"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Status    int    `json:"status"`
	IP        string `json:"ip"`
}

// Record appends an entry to the audit log (At is set when empty).
func (s *UserStore) Record(e AuditEntry) error {
	if e.At == "" {
		e.At = formatTime(s.now())
	}
	if e.UserLabel == "" {
		e.UserLabel = "guest"
	}
	_, err := s.db.Exec(`INSERT INTO audit_log (at, user_id, user_label, action, target, status, ip) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.At, e.UserID, truncate(e.UserLabel, 320), truncate(e.Action, 200), truncate(e.Target, 1024), e.Status, e.IP)
	return err
}

// Page size of ListAudit: default when unset or not positive, cap above.
const (
	auditDefaultLimit = 100
	auditMaxLimit     = 500
)

// auditLimit turns a requested page size into one ListAudit serves: not
// positive (unset, garbage) → 100, over 500 → 500.
func auditLimit(limit int) int {
	switch {
	case limit <= 0:
		return auditDefaultLimit
	case limit > auditMaxLimit:
		return auditMaxLimit
	}
	return limit
}

// ListAudit returns up to limit entries (see auditLimit), newest first;
// before > 0 lists entries older than that id (the next page).
func (s *UserStore) ListAudit(limit int, before int64) ([]AuditEntry, error) {
	limit = auditLimit(limit)
	q := `SELECT id, at, user_id, user_label, action, target, status, ip FROM audit_log`
	args := []any{}
	if before > 0 {
		q += ` WHERE id < ?`
		args = append(args, before)
	}
	q += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.At, &e.UserID, &e.UserLabel, &e.Action, &e.Target, &e.Status, &e.IP); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Prune deletes audit entries older than auditBefore and sessions that
// expired or were revoked before sessionsBefore.
func (s *UserStore) Prune(auditBefore, sessionsBefore time.Time) error {
	if _, err := s.db.Exec(`DELETE FROM audit_log WHERE at < ?`, formatTime(auditBefore)); err != nil {
		return err
	}
	cut := formatTime(sessionsBefore)
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < ? OR revoked_at < ?`, cut, cut)
	return err
}

func randomID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
