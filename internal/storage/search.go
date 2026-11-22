package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// SearchFunc returns the text full-text search finds an event by, "" for none.
type SearchFunc func(eventType string, payload json.RawMessage) string

// minTrigramTerm is the shortest word the trigram index can match; shorter
// words are matched with LIKE over the indexed text.
const minTrigramTerm = 3

// SetSearchFunc makes Store and UpdatePayload index the text fn returns for
// GET /api/events?q=. Set it at startup; nil indexes nothing.
func (s *SQLiteStore) SetSearchFunc(fn SearchFunc) {
	s.search = fn
}

// InitSearch creates the trigger that removes an event's search rows with
// the event (see InitFacets).
func InitSearch(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TRIGGER IF NOT EXISTS events_delete_search
AFTER DELETE ON events
BEGIN
    DELETE FROM event_search WHERE rowid = (SELECT id FROM event_search_ids WHERE uuid = old.uuid);
    DELETE FROM event_search_ids WHERE uuid = old.uuid;
END`)
	return err
}

// writeSearch replaces the indexed text of one event inside tx.
func writeSearch(ctx context.Context, tx *sql.Tx, uuid, text string) error {
	var id int64
	err := tx.QueryRowContext(ctx, `SELECT id FROM event_search_ids WHERE uuid = ?`, uuid).Scan(&id)
	switch {
	case err == sql.ErrNoRows:
		if strings.TrimSpace(text) == "" {
			return nil
		}
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO event_search_ids (uuid) VALUES (?) RETURNING id`, uuid).Scan(&id); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if _, err := tx.ExecContext(ctx, `DELETE FROM event_search WHERE rowid = ?`, id); err != nil {
			return err
		}
		if strings.TrimSpace(text) == "" {
			_, err := tx.ExecContext(ctx, `DELETE FROM event_search_ids WHERE id = ?`, id)
			return err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO event_search (rowid, body) VALUES (?, ?)`, id, text)
	return err
}

// searchCondition turns a search into a WHERE condition on col (the events
// uuid column): every word must occur in the event's indexed text.
func searchCondition(col, search string) (string, []any, bool) {
	var phrases []string
	var likes []string
	var args []any
	for _, w := range strings.Fields(search) {
		if utf8.RuneCountInString(w) >= minTrigramTerm {
			phrases = append(phrases, `"`+strings.ReplaceAll(w, `"`, `""`)+`"`)
		} else {
			likes = append(likes, `body LIKE ? ESCAPE '\'`)
			args = append(args, "%"+likeEscape(w)+"%")
		}
	}
	if len(phrases) == 0 && len(likes) == 0 {
		return "", nil, false
	}
	var conds []string
	if len(phrases) > 0 {
		conds = append(conds, "event_search MATCH ?")
		args = append([]any{strings.Join(phrases, " AND ")}, args...)
	}
	conds = append(conds, likes...)
	return col + ` IN (SELECT i.uuid FROM event_search_ids i WHERE i.id IN (SELECT rowid FROM event_search WHERE ` +
		strings.Join(conds, " AND ") + `))`, args, true
}

func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
