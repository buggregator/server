package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

// FacetFunc returns the filterable values of a payload of an event type.
type FacetFunc func(eventType string, payload json.RawMessage) []event.Facet

// facetBackfillBatch is how many old events one backfill step indexes; each
// step is one short write transaction, and facetBackfillPause between steps
// lets the feed and ingestion through (without it, on 120k events, the feed
// waited 2-9 s for the database while the backfill ran).
const facetBackfillBatch = 200

var facetBackfillPause = 50 * time.Millisecond

// MaxFacetValuesPerKey caps the values FacetCounts returns for one key.
const MaxFacetValuesPerKey = 50

// SetFacetFunc makes Store and UpdatePayload index the facets fn returns.
// Set it at startup, before the store is used; nil indexes nothing.
func (s *SQLiteStore) SetFacetFunc(fn FacetFunc) {
	s.facets = fn
}

// InitFacets creates the trigger that removes an event's facets with the
// event, whatever path deletes it (store, retention, module cleanup). It is
// not in the migration: the migration splitter cuts at ';'.
func InitFacets(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TRIGGER IF NOT EXISTS events_delete_facets
AFTER DELETE ON events
BEGIN
    DELETE FROM event_facets WHERE uuid = old.uuid;
END`)
	return err
}

// writeFacets replaces the facets of one event inside tx.
func writeFacets(ctx context.Context, tx *sql.Tx, uuid string, facets []event.Facet) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM event_facets WHERE uuid = ?`, uuid); err != nil {
		return err
	}
	for _, f := range facets {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO event_facets (uuid, key, value) VALUES (?, ?, ?)`,
			uuid, f.Key, f.Value); err != nil {
			return err
		}
	}
	return nil
}

// facetConditions turns filters into WHERE conditions on events.uuid; col is
// the uuid column of the outer query. skipKey leaves one key out (facet
// counts of a key ignore that key's own filter, so its other values stay
// visible).
func facetConditions(col string, filters []event.FacetFilter, skipKey string) ([]string, []any) {
	var conds []string
	var args []any
	for _, f := range filters {
		if f.Key == skipKey || len(f.Values) == 0 {
			continue
		}
		ph := strings.TrimSuffix(strings.Repeat("?,", len(f.Values)), ",")
		op := "IN"
		if f.Not {
			op = "NOT IN"
		}
		conds = append(conds, fmt.Sprintf(
			"%s %s (SELECT uuid FROM event_facets WHERE key = ? AND value IN (%s))", col, op, ph))
		args = append(args, f.Key)
		for _, v := range f.Values {
			args = append(args, event.NormalizeFacetValue(f.Key, v))
		}
	}
	return conds, args
}

// eventConditions builds the WHERE conditions of FindOptions on the events
// table aliased as alias ("" for none).
func eventConditions(opts event.FindOptions, alias, skipKey string) ([]string, []any) {
	col := func(c string) string {
		if alias == "" {
			return c
		}
		return alias + "." + c
	}
	var conds []string
	var args []any
	if opts.Type != "" {
		conds = append(conds, col("type")+" = ?")
		args = append(args, opts.Type)
	}
	if opts.Project != "" {
		conds = append(conds, col("project")+" = ?")
		args = append(args, opts.Project)
	}
	// Window bounds are compared numerically: timestamp is stored as text
	// (fmt.Sprintf("%.6f")), and a string comparison would break as soon as the
	// number of digits in unix seconds changes.
	if opts.From > 0 {
		conds = append(conds, "CAST("+col("timestamp")+" AS REAL) >= ?")
		args = append(args, opts.From)
	}
	if opts.To > 0 {
		conds = append(conds, "CAST("+col("timestamp")+" AS REAL) <= ?")
		args = append(args, opts.To)
	}
	fc, fa := facetConditions(col("uuid"), opts.Filters, skipKey)
	conds, args = append(conds, fc...), append(args, fa...)
	if sc, sa, ok := searchCondition(col("uuid"), opts.Search); ok {
		conds, args = append(conds, sc), append(args, sa...)
	}
	return conds, args
}

// FacetCounts returns, per facet key, its values with the number of events
// matching opts (type, project, window, filters), most frequent first, at
// most perKey values per key (0 = MaxFacetValuesPerKey). A key's counts
// ignore the filter on that same key, so picking one host still shows the
// others. Limit and Offset of opts are ignored.
func (s *SQLiteStore) FacetCounts(ctx context.Context, opts event.FindOptions, perKey int) (map[string][]event.FacetCount, error) {
	if perKey <= 0 {
		perKey = MaxFacetValuesPerKey
	}
	result := map[string][]event.FacetCount{}

	// One query for keys without their own filter, one per filtered key.
	filtered := map[string]bool{}
	for _, f := range opts.Filters {
		if len(f.Values) > 0 {
			filtered[f.Key] = true
		}
	}
	add := func(skipKey string, onlyKey string) error {
		conds, args := eventConditions(opts, "e", skipKey)
		if onlyKey != "" {
			conds = append(conds, "f.key = ?")
			args = append(args, onlyKey)
		} else if len(filtered) > 0 {
			keys := make([]string, 0, len(filtered))
			for k := range filtered {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			conds = append(conds, "f.key NOT IN ("+strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")+")")
			for _, k := range keys {
				args = append(args, k)
			}
		}
		query := `SELECT f.key, f.value, COUNT(*) AS n FROM event_facets f JOIN events e ON e.uuid = f.uuid`
		if len(conds) > 0 {
			query += " WHERE " + strings.Join(conds, " AND ")
		}
		query += " GROUP BY f.key, f.value ORDER BY f.key, n DESC, f.value"
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var key, value string
			var n int
			if err := rows.Scan(&key, &value, &n); err != nil {
				return err
			}
			if len(result[key]) < perKey {
				result[key] = append(result[key], event.FacetCount{Value: value, Count: n})
			}
		}
		return rows.Err()
	}

	if err := add("", ""); err != nil {
		return nil, err
	}
	for key := range filtered {
		if err := add(key, key); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// BackfillFacets indexes the facets of events stored before facets existed.
func (s *SQLiteStore) BackfillFacets(ctx context.Context) error {
	if s.facets == nil {
		return nil
	}
	return s.backfill(ctx, "facets", "event_facets_backfill", func(ctx context.Context, tx *sql.Tx, uuid, typ, payload string) error {
		return writeFacets(ctx, tx, uuid, s.facets(typ, json.RawMessage(payload)))
	})
}

// BackfillSearch indexes the search text of events stored before search existed.
func (s *SQLiteStore) BackfillSearch(ctx context.Context) error {
	if s.search == nil {
		return nil
	}
	return s.backfill(ctx, "search", "event_search_backfill", func(ctx context.Context, tx *sql.Tx, uuid, typ, payload string) error {
		return writeSearch(ctx, tx, uuid, s.search(typ, json.RawMessage(payload)))
	})
}

// indexFunc writes one stored event's index rows inside tx.
type indexFunc func(ctx context.Context, tx *sql.Tx, uuid, typ, payload string) error

// backfill indexes the events stored before an index existed (rowid up to
// the cutoff the migration recorded in progressTable), one short
// transaction per batch, until done or ctx ends. Progress is saved, so a
// restart continues.
func (s *SQLiteStore) backfill(ctx context.Context, name, progressTable string, write indexFunc) error {
	var cursor, cutoff int64
	err := s.db.QueryRowContext(ctx,
		`SELECT cursor, cutoff FROM `+progressTable+` WHERE id = 1`).Scan(&cursor, &cutoff)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if cursor >= cutoff {
		return nil
	}
	started, total := time.Now(), 0
	slog.Info(name+": indexing stored events in the background", "from_rowid", cursor, "to_rowid", cutoff)
	for cursor < cutoff {
		if err := ctx.Err(); err != nil {
			return err
		}
		next, n, err := s.backfillBatch(ctx, progressTable, write, cursor, cutoff)
		if err != nil {
			return err
		}
		cursor = next
		total += n
		if cursor < cutoff && facetBackfillPause > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(facetBackfillPause):
			}
		}
	}
	slog.Info(name+": stored events indexed", "events", total, "took", time.Since(started).Round(time.Millisecond))
	return nil
}

// backfillBatch indexes one batch after cursor and saves the new cursor, in
// one transaction: a Ray merge of the same event between the read and the
// write cannot be overwritten by the stale payload.
func (s *SQLiteStore) backfillBatch(ctx context.Context, progressTable string, write indexFunc, cursor, cutoff int64) (int64, int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return cursor, 0, err
	}
	defer tx.Rollback() //nolint:errcheck // no-op after Commit

	rows, err := tx.QueryContext(ctx,
		`SELECT rowid, uuid, type, payload FROM events WHERE rowid > ? AND rowid <= ? ORDER BY rowid LIMIT ?`,
		cursor, cutoff, facetBackfillBatch)
	if err != nil {
		return cursor, 0, err
	}
	type item struct {
		uuid, typ string
		payload   string
	}
	var items []item
	next := cutoff // no rows left: done
	for rows.Next() {
		var rowid int64
		var it item
		if err := rows.Scan(&rowid, &it.uuid, &it.typ, &it.payload); err != nil {
			rows.Close()
			return cursor, 0, err
		}
		items = append(items, it)
		next = rowid
	}
	if err := rows.Close(); err != nil {
		return cursor, 0, err
	}
	if len(items) < facetBackfillBatch {
		next = cutoff
	}
	for _, it := range items {
		if err := write(ctx, tx, it.uuid, it.typ, it.payload); err != nil {
			return cursor, 0, err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE `+progressTable+` SET cursor = ? WHERE id = 1`, next); err != nil {
		return cursor, 0, err
	}
	if err := tx.Commit(); err != nil {
		return cursor, 0, err
	}
	return next, len(items), nil
}
