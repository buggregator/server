package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// nPlusOneThreshold is how many spans of one transaction must run the same
// normalized query before the trace reports it as an N+1 pattern.
const nPlusOneThreshold = 5

var (
	sqlStringLit   = regexp.MustCompile(`'(?:[^']|'')*'`)
	sqlPgParam     = regexp.MustCompile(`\$\d+\b`)
	sqlNumberLit   = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)
	sqlInList      = regexp.MustCompile(`(?i)\bIN\s*\(\s*\?(?:\s*,\s*\?)*\s*\)`)
	sqlValuesList  = regexp.MustCompile(`(?i)\bVALUES\s*\(\s*\?(?:\s*,\s*\?)*\s*\)(?:\s*,\s*\(\s*\?(?:\s*,\s*\?)*\s*\))*`)
	sqlWhitespaces = regexp.MustCompile(`\s+`)
)

// normalizeSQL turns a query into its shape: string and number literals and
// numbered placeholders become ?, an IN list or a VALUES list of any length
// becomes one ?, whitespace collapses. Two queries that differ only in their
// parameters normalize to the same text.
func normalizeSQL(q string) string {
	q = sqlStringLit.ReplaceAllString(q, "?")
	q = sqlPgParam.ReplaceAllString(q, "?")
	q = sqlNumberLit.ReplaceAllString(q, "?")
	q = sqlInList.ReplaceAllString(q, "IN (?)")
	q = sqlValuesList.ReplaceAllString(q, "VALUES (?)")
	q = sqlWhitespaces.ReplaceAllString(q, " ")
	return strings.TrimSpace(q)
}

// nPlusOneSpan is the part of a database span the detector reads.
type nPlusOneSpan struct {
	SpanID        string
	TransactionID string
	Description   string
	DurationMS    float64
	Attributes    string // JSON object, may be empty
}

// detectNPlusOne groups database spans by transaction and normalized query
// and returns the groups of at least nPlusOneThreshold spans, most repeated
// first. origin is the code place (code.filepath:code.lineno) of the first
// span that has one; sentry-laravel writes it only for queries slower than
// SENTRY_TRACE_SQL_ORIGIN_THRESHOLD_MS (100 ms by default).
func detectNPlusOne(spans []nPlusOneSpan) []map[string]any {
	type group struct {
		txn, query string
		count      int
		totalMS    float64
		origin     string
		spanIDs    []string
		first      int
	}
	groups := map[string]*group{}
	for i, s := range spans {
		q := normalizeSQL(s.Description)
		if q == "" {
			continue
		}
		key := s.TransactionID + "\x00" + q
		g := groups[key]
		if g == nil {
			g = &group{txn: s.TransactionID, query: q, first: i}
			groups[key] = g
		}
		g.count++
		g.totalMS += s.DurationMS
		g.spanIDs = append(g.spanIDs, s.SpanID)
		if g.origin == "" {
			g.origin = spanOrigin(s.Attributes)
		}
	}

	var found []*group
	for _, g := range groups {
		if g.count >= nPlusOneThreshold {
			found = append(found, g)
		}
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].count != found[j].count {
			return found[i].count > found[j].count
		}
		if found[i].totalMS != found[j].totalMS {
			return found[i].totalMS > found[j].totalMS
		}
		return found[i].first < found[j].first
	})

	result := make([]map[string]any, 0, len(found))
	for _, g := range found {
		result = append(result, map[string]any{
			"transaction_id": nullIfEmpty(g.txn),
			"query":          g.query,
			"count":          g.count,
			"total_ms":       math.Round(g.totalMS*100) / 100,
			"origin":         nullIfEmpty(g.origin),
			"span_ids":       g.spanIDs,
		})
	}
	return result
}

// spanOrigin reads code.filepath and code.lineno from a span's attributes.
func spanOrigin(attrsJSON string) string {
	if attrsJSON == "" {
		return ""
	}
	var attrs map[string]any
	if json.Unmarshal([]byte(attrsJSON), &attrs) != nil {
		return ""
	}
	file, _ := attrs["code.filepath"].(string)
	if file == "" {
		return ""
	}
	switch line := attrs["code.lineno"].(type) {
	case float64:
		return fmt.Sprintf("%s:%d", file, int(line))
	case string:
		if line != "" {
			return file + ":" + line
		}
	}
	return file
}

// loadNPlusOne runs detectNPlusOne over a trace's database spans.
func loadNPlusOne(db *sql.DB, traceID string, projectClause string, projectArgs []any) []map[string]any {
	rows, err := db.Query(
		`SELECT span_id, transaction_id, description, start_ts, end_ts, duration_ms, attributes
		FROM sentry_spans WHERE trace_id = ? AND peer_type = 'db'`+projectClause+` ORDER BY start_ts`,
		append([]any{traceID}, projectArgs...)...,
	)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var spans []nPlusOneSpan
	for rows.Next() {
		var (
			spanID             string
			txnID, desc, attrs sql.NullString
			startTS, endTS     sql.NullString
			durationMS         sql.NullInt64
		)
		if rows.Scan(&spanID, &txnID, &desc, &startTS, &endTS, &durationMS, &attrs) != nil {
			continue
		}
		// duration_ms is whole milliseconds, and a repeated query often takes
		// less than one: sum the precise span times when both are known.
		dur := float64(durationMS.Int64)
		if st, err := parseRFC3339(startTS.String); err == nil {
			if et, err := parseRFC3339(endTS.String); err == nil && !et.Before(st) {
				dur = float64(et.Sub(st).Microseconds()) / 1000
			}
		}
		spans = append(spans, nPlusOneSpan{
			SpanID:        spanID,
			TransactionID: txnID.String,
			Description:   desc.String,
			DurationMS:    dur,
			Attributes:    attrs.String,
		})
	}
	return detectNPlusOne(spans)
}
