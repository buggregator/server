package sentry

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// storeTransaction stores a parsed transaction with its spans, upserting the
// trace registry entry. Returns the transaction's generated UUID.
func storeTransaction(db *sql.DB, txn *Transaction, payload json.RawMessage, project string) (string, error) {
	project = projectKey(project)
	dbTx, err := db.Begin()
	if err != nil {
		return "", err
	}
	defer dbTx.Rollback()

	traceID := ""
	if txn.Contexts != nil && txn.Contexts.Trace != nil {
		traceID = txn.Contexts.Trace.TraceID
	}

	if traceID == "" {
		// Try to get trace_id from spans.
		for _, s := range txn.Spans {
			if s.TraceID != "" {
				traceID = s.TraceID
				break
			}
		}
	}

	if traceID == "" {
		return "", fmt.Errorf("transaction has no trace_id")
	}

	spanCount := len(txn.Spans)

	// Upsert sentry_traces.
	_, err = dbTx.Exec(
		`INSERT INTO sentry_traces (trace_id, span_count)
		VALUES (?, ?)
		ON CONFLICT(trace_id) DO UPDATE SET
			last_seen = datetime('now'),
			span_count = sentry_traces.span_count + excluded.span_count`,
		traceID, spanCount,
	)
	if err != nil {
		return "", fmt.Errorf("upsert sentry_traces: %w", err)
	}

	// Insert transaction.
	txnID := event.GenerateUUID()
	startTS := parseTimestamp(txn.StartTime.Number())
	endTS := parseTimestamp(txn.Timestamp.Number())
	durationMS := computeDurationMS(txn.StartTime.Number(), txn.Timestamp.Number())

	op := txn.Op
	status := txn.Status
	// Fallback: SDK v3/v4 send op/status only in contexts.trace.
	if txn.Contexts != nil && txn.Contexts.Trace != nil {
		if op == "" {
			op = txn.Contexts.Trace.Op
		}
		if status == "" {
			status = txn.Contexts.Trace.Status
		}
	}

	var measurements *string
	if txn.Measurements != nil {
		s := string(txn.Measurements)
		measurements = &s
	}

	_, err = dbTx.Exec(
		`INSERT INTO sentry_transactions
			(id, event_id, trace_id, transaction_name, op, status, start_ts, end_ts, duration_ms, environment, release, measurements, payload, project_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		txnID,
		txn.EventID,
		traceID,
		txn.Transaction,
		nullIfEmpty(op),
		nullIfEmpty(status),
		startTS,
		endTS,
		durationMS,
		nullIfEmpty(txn.Environment),
		nullIfEmpty(txn.Release),
		measurements,
		string(payload),
		project,
	)
	if err != nil {
		return "", fmt.Errorf("insert sentry_transactions: %w", err)
	}

	// Insert spans.
	errorCount := 0
	if len(txn.Spans) > 0 {
		stmt, err := dbTx.Prepare(
			`INSERT INTO sentry_spans
				(id, span_id, transaction_id, trace_id, parent_span_id, op, description, status, start_ts, end_ts, duration_ms, service_name, peer_address, peer_type, is_error, attributes, project_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		)
		if err != nil {
			return "", fmt.Errorf("prepare sentry_spans: %w", err)
		}
		defer stmt.Close()

		for _, span := range txn.Spans {
			spanUUID := event.GenerateUUID()
			peerType, peerAddress := classifySpan(span)
			serviceName := extractServiceName(span, txn.SDK)
			sStartTS := parseTimestamp(span.StartTimestamp.Number())
			sEndTS := parseTimestamp(span.Timestamp.Number())
			sDuration := computeDurationMS(span.StartTimestamp.Number(), span.Timestamp.Number())

			isError := 0
			if isErrorStatus(span.Status) {
				isError = 1
				errorCount++
			}

			var attrs *string
			if span.Data != nil {
				b, _ := json.Marshal(span.Data)
				s := string(b)
				attrs = &s
			}

			_, err = stmt.Exec(
				spanUUID,
				span.SpanID,
				txnID,
				traceID,
				nullIfEmpty(span.ParentSpanID),
				nullIfEmpty(span.Op),
				nullIfEmpty(span.Description),
				nullIfEmpty(span.Status),
				sStartTS,
				sEndTS,
				sDuration,
				nullIfEmpty(serviceName),
				nullIfEmpty(peerAddress),
				nullIfEmpty(peerType),
				isError,
				attrs,
				project,
			)
			if err != nil {
				return "", fmt.Errorf("insert sentry_spans: %w", err)
			}
		}
	}

	// Update error count on trace.
	if errorCount > 0 {
		_, err = dbTx.Exec(
			`UPDATE sentry_traces SET error_count = error_count + ? WHERE trace_id = ?`,
			errorCount, traceID,
		)
		if err != nil {
			return "", fmt.Errorf("update sentry_traces error_count: %w", err)
		}
	}

	return txnID, dbTx.Commit()
}

// storeSpansV2 stores standalone span items (Sentry Span v2 format),
// upserting the trace registry entry.
func storeSpansV2(db *sql.DB, spans []RawSpan, sdk *SDK, project string) error {
	project = projectKey(project)
	if len(spans) == 0 {
		return nil
	}

	dbTx, err := db.Begin()
	if err != nil {
		return err
	}
	defer dbTx.Rollback()

	// Group by trace_id for upserts.
	traceSpans := make(map[string][]RawSpan)
	for _, s := range spans {
		traceSpans[s.TraceID] = append(traceSpans[s.TraceID], s)
	}

	stmt, err := dbTx.Prepare(
		`INSERT INTO sentry_spans
			(id, span_id, transaction_id, trace_id, parent_span_id, op, description, status, start_ts, end_ts, duration_ms, service_name, peer_address, peer_type, is_error, attributes, project_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare sentry_spans: %w", err)
	}
	defer stmt.Close()

	for traceID, group := range traceSpans {
		traceErrors := 0

		// A segment span is the root of a streamed trace (JS SDK v10+ span
		// streaming): it becomes the transaction, as a transaction item's root
		// would, and is not a span row of its own.
		children := 0
		for _, span := range group {
			if !span.IsSegment {
				children++
			}
		}

		// Upsert trace.
		_, err = dbTx.Exec(
			`INSERT INTO sentry_traces (trace_id, span_count)
			VALUES (?, ?)
			ON CONFLICT(trace_id) DO UPDATE SET
				last_seen = datetime('now'),
				span_count = sentry_traces.span_count + excluded.span_count`,
			traceID, children,
		)
		if err != nil {
			return fmt.Errorf("upsert sentry_traces: %w", err)
		}

		for _, span := range group {
			if span.IsSegment {
				if err := storeSegment(dbTx, traceID, span, project); err != nil {
					return err
				}
				continue
			}
			spanUUID := event.GenerateUUID()
			peerType, peerAddress := classifySpan(span)
			serviceName := extractServiceName(span, sdk)
			sStartTS := parseTimestamp(span.StartTimestamp.Number())
			sEndTS := parseTimestamp(span.Timestamp.Number())
			sDuration := computeDurationMS(span.StartTimestamp.Number(), span.Timestamp.Number())

			isError := 0
			if isErrorStatus(span.Status) {
				isError = 1
				traceErrors++
			}

			var attrs *string
			if span.Data != nil {
				b, _ := json.Marshal(span.Data)
				s := string(b)
				attrs = &s
			}

			_, err = stmt.Exec(
				spanUUID,
				span.SpanID,
				nil, // transaction_id — standalone spans have no transaction
				traceID,
				nullIfEmpty(span.ParentSpanID),
				nullIfEmpty(span.Op),
				nullIfEmpty(span.Description),
				nullIfEmpty(span.Status),
				sStartTS,
				sEndTS,
				sDuration,
				nullIfEmpty(serviceName),
				nullIfEmpty(peerAddress),
				nullIfEmpty(peerType),
				isError,
				attrs,
				project,
			)
			if err != nil {
				return fmt.Errorf("insert sentry_spans: %w", err)
			}
		}

		if traceErrors > 0 {
			_, err = dbTx.Exec(
				`UPDATE sentry_traces SET error_count = error_count + ? WHERE trace_id = ?`,
				traceErrors, traceID,
			)
			if err != nil {
				return fmt.Errorf("update sentry_traces error_count: %w", err)
			}
		}
	}

	return dbTx.Commit()
}

// storeSegment stores a segment span as a transaction. Its event_id is made
// from the trace and span ids, so a segment sent twice is stored once.
func storeSegment(dbTx *sql.Tx, traceID string, span RawSpan, project string) error {
	payload, _ := json.Marshal(span)
	_, err := dbTx.Exec(
		`INSERT OR IGNORE INTO sentry_transactions
			(id, event_id, trace_id, transaction_name, op, status, start_ts, end_ts, duration_ms, environment, release, measurements, payload, project_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, ?, ?)`,
		event.GenerateUUID(),
		traceID+"-"+span.SpanID,
		traceID,
		span.Description,
		nullIfEmpty(span.Op),
		nullIfEmpty(span.Status),
		parseTimestamp(span.StartTimestamp.Number()),
		parseTimestamp(span.Timestamp.Number()),
		computeDurationMS(span.StartTimestamp.Number(), span.Timestamp.Number()),
		nullIfEmpty(span.Data["sentry.environment"]),
		nullIfEmpty(span.Data["sentry.release"]),
		string(payload),
		project,
	)
	if err != nil {
		return fmt.Errorf("insert segment transaction: %w", err)
	}
	return nil
}

// isErrorStatus reports whether a span status means failure: the v1 span
// statuses internal_error/unknown_error, or "error" from Span v2 items.
func isErrorStatus(status string) bool {
	return status == "internal_error" || status == "unknown_error" || status == "error"
}

// computeDurationMS calculates the duration in milliseconds between two JSON number timestamps.
func computeDurationMS(start, end json.Number) *int {
	if start == "" || end == "" {
		return nil
	}
	s, err1 := strconv.ParseFloat(string(start), 64)
	e, err2 := strconv.ParseFloat(string(end), 64)
	if err1 != nil || err2 != nil {
		return nil
	}
	ms := int((e - s) * 1000)
	if ms < 0 {
		ms = 0
	}
	return &ms
}

// formatTimestamp converts an epoch-seconds float to RFC3339.
func formatTimestamp(epochSeconds float64) string {
	sec := int64(epochSeconds)
	nsec := int64((epochSeconds - float64(sec)) * 1e9)
	return time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
}
