package sentry

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

type handler struct {
	db *sql.DB
}

func (h *handler) Priority() int { return 10 }

func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if r.Header.Get("X-Buggregator-Detected-Type") == "sentry" {
		return true
	}
	if r.Header.Get("X-Sentry-Auth") != "" {
		return true
	}
	path := r.URL.Path
	isSentryStore := strings.HasSuffix(path, "/store") && !strings.Contains(path, "/profiler/")
	return isSentryStore || strings.HasSuffix(path, "/envelope")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Sentry SDKs often send gzip or deflate compressed payloads.
	body = decompress(body, r.Header.Get("Content-Encoding"))

	// Extract project from path: /api/{project}/store
	project := extractProject(r.URL.Path)

	// Try parsing as plain JSON first (legacy /store format).
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) == nil {
		uuid := ""
		if id, ok := parsed["event_id"].(string); ok {
			uuid = id
		}

		// Store structured data if DB is available.
		h.storeJSONEvent(body, project)

		// Detect transactions (have spans + start_timestamp but no exception).
		// Transactions are stored in structured tables but should not appear
		// as empty cards in the global event feed.
		if isTransaction(parsed) {
			return nil, nil
		}

		return &event.Incoming{
			UUID:    uuid,
			Type:    "sentry",
			Payload: json.RawMessage(body),
			Project: project,
		}, nil
	}

	// Not valid JSON — parse as envelope format.
	return h.handleEnvelope(body, project)
}

// storeJSONEvent stores a plain JSON error event in structured tables.
func (h *handler) storeJSONEvent(body []byte, project string) {
	if h.db == nil {
		return
	}

	var ev ErrorEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		return
	}

	if _, err := storeErrorEvent(h.db, &ev, body, project); err != nil {
		slog.Warn("sentry: failed to store structured error event", "err", err)
	}
}

// handleEnvelope parses Sentry envelope format and routes each item
// to the appropriate storage function.
func (h *handler) handleEnvelope(body []byte, project string) (*event.Incoming, error) {
	envHeader, items, err := parseEnvelopeItems(body)
	if err != nil {
		return nil, err
	}

	// Track the primary item for the canonical event.
	var primaryUUID string
	var primaryPayload json.RawMessage
	var primaryType string

	if envHeader.EventID != "" {
		primaryUUID = envHeader.EventID
	}

	for _, item := range items {
		switch item.Type {
		case "event":
			uuid, payload := h.handleEventItem(item, envHeader.EventID, project)
			if uuid != "" {
				primaryUUID = uuid
			}
			if payload != nil {
				primaryPayload = payload
				primaryType = "sentry"
			}

		case "transaction":
			// Transactions are stored in structured tables (sentry_transactions, sentry_spans)
			// but should NOT appear in the global event feed as empty sentry cards.
			h.handleTransactionItem(item)

		case "spans":
			h.handleSpansItem(item)

		case "log":
			h.handleLogItem(item)

		case "session", "sessions", "client_report", "metric_buckets", "check_in",
			"user_report", "attachment", "replay_event", "replay_recording", "statsd":
			// Silently discard known non-essential item types.

		default:
			slog.Debug("sentry: ignoring unknown envelope item type", "type", item.Type)
		}
	}

	// Validate the primary payload is valid JSON; if not, discard it.
	if primaryPayload != nil && !json.Valid(primaryPayload) {
		primaryPayload = nil
	}

	// If no primary payload was extracted, fall back to envelope wrapping.
	// Skip transaction/spans/log items — they are stored in structured tables only.
	if primaryPayload == nil {
		primaryType = "sentry"
		for _, item := range items {
			if item.Type == "transaction" || item.Type == "spans" || item.Type == "log" {
				continue
			}
			if json.Valid([]byte(item.Payload)) {
				primaryPayload = item.Payload
				break
			}
		}
	}

	// If still no payload, the envelope only contained non-feed items
	// (transactions, spans, logs). Don't create a canonical event.
	if primaryPayload == nil {
		return nil, nil
	}

	return &event.Incoming{
		UUID:    primaryUUID,
		Type:    primaryType,
		Payload: primaryPayload,
		Project: project,
	}, nil
}

// handleEventItem processes an "event" envelope item (error/message event).
// envelopeEventID is the event_id from the envelope header — Sentry SDK v4+
// omits event_id from the item payload, so we inject it when missing.
func (h *handler) handleEventItem(item EnvelopeItem, envelopeEventID string, project string) (string, json.RawMessage) {
	var ev ErrorEvent
	if err := json.Unmarshal(item.Payload, &ev); err != nil {
		slog.Warn("sentry: failed to parse error event", "err", err)
		return "", item.Payload
	}

	// Sentry SDK v4+ sends event_id only in the envelope header, not in
	// the item payload. Inject it so downstream consumers (frontend, API)
	// always see it in the payload.
	uuid := ev.EventID
	payload := item.Payload
	if uuid == "" && envelopeEventID != "" {
		uuid = envelopeEventID
		ev.EventID = envelopeEventID
		payload = injectEventID(payload, envelopeEventID)
	}

	if h.db != nil {
		if _, err := storeErrorEvent(h.db, &ev, payload, project); err != nil {
			slog.Warn("sentry: failed to store structured error event", "err", err)
		}
	}

	return uuid, payload
}

// handleTransactionItem processes a "transaction" envelope item.
func (h *handler) handleTransactionItem(item EnvelopeItem) (string, json.RawMessage) {
	var txn Transaction
	if err := json.Unmarshal(item.Payload, &txn); err != nil {
		slog.Warn("sentry: failed to parse transaction", "err", err)
		return "", item.Payload
	}

	uuid := txn.EventID

	if h.db != nil {
		if _, err := storeTransaction(h.db, &txn, item.Payload); err != nil {
			slog.Warn("sentry: failed to store structured transaction", "err", err)
		}
	}

	return uuid, item.Payload
}

// handleSpansItem processes a "spans" envelope item (Span v2 format).
func (h *handler) handleSpansItem(item EnvelopeItem) {
	if h.db == nil {
		return
	}

	// Spans v2 can be either {"items": [...]} or a bare array [...].
	var envelope SpansEnvelope
	if err := json.Unmarshal(item.Payload, &envelope); err == nil && len(envelope.Items) > 0 {
		if err := storeSpansV2(h.db, envelope.Items, nil); err != nil {
			slog.Warn("sentry: failed to store spans v2", "err", err)
		}
		return
	}

	// Try as bare array.
	var spans []RawSpan
	if err := json.Unmarshal(item.Payload, &spans); err == nil && len(spans) > 0 {
		if err := storeSpansV2(h.db, spans, nil); err != nil {
			slog.Warn("sentry: failed to store spans v2 (array)", "err", err)
		}
	}
}

// handleLogItem processes a "log" envelope item.
func (h *handler) handleLogItem(item EnvelopeItem) {
	if h.db == nil {
		return
	}

	// Log items can be {"items": [...]} or a bare array.
	var envelope LogEnvelope
	if err := json.Unmarshal(item.Payload, &envelope); err == nil && len(envelope.Items) > 0 {
		if err := storeLogs(h.db, envelope.Items); err != nil {
			slog.Warn("sentry: failed to store logs", "err", err)
		}
		return
	}

	var logs []LogRecord
	if err := json.Unmarshal(item.Payload, &logs); err == nil && len(logs) > 0 {
		if err := storeLogs(h.db, logs); err != nil {
			slog.Warn("sentry: failed to store logs (array)", "err", err)
		}
	}
}

// injectEventID adds "event_id" to a JSON payload that lacks it.
// Used to normalize Sentry SDK v4+ payloads which omit event_id from item bodies.
func injectEventID(payload json.RawMessage, eventID string) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(payload, &obj); err != nil {
		return payload
	}
	obj["event_id"] = json.RawMessage(`"` + eventID + `"`)
	result, err := json.Marshal(obj)
	if err != nil {
		return payload
	}
	return result
}

// decompress handles gzip and deflate Content-Encoding.
// Also auto-detects gzip/zlib by magic bytes if header is missing.
func decompress(data []byte, encoding string) []byte {
	if len(data) == 0 {
		return data
	}

	// Try by Content-Encoding header first.
	switch strings.ToLower(encoding) {
	case "gzip":
		if d, err := decompressGzip(data); err == nil {
			return d
		}
	case "deflate":
		if d, err := decompressZlib(data); err == nil {
			return d
		}
	}

	// Auto-detect by magic bytes (Sentry SDKs sometimes omit the header).
	if len(data) >= 2 {
		// Gzip magic: 0x1f 0x8b
		if data[0] == 0x1f && data[1] == 0x8b {
			if d, err := decompressGzip(data); err == nil {
				return d
			}
		}
		// Zlib magic: 0x78 (0x01, 0x5e, 0x9c, 0xda)
		if data[0] == 0x78 {
			if d, err := decompressZlib(data); err == nil {
				return d
			}
		}
	}

	return data
}

func decompressGzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

// isTransaction detects Sentry transaction payloads that should be stored
// in structured tables but NOT appear in the global event feed.
// Transactions have spans and start_timestamp but typically no exception data.
func isTransaction(parsed map[string]any) bool {
	_, hasSpans := parsed["spans"]
	_, hasStartTS := parsed["start_timestamp"]
	if !hasSpans || !hasStartTS {
		return false
	}

	// If it has exception data, it's an error event with trace context — keep it.
	if exc, ok := parsed["exception"].(map[string]any); ok {
		if vals, ok := exc["values"].([]any); ok && len(vals) > 0 {
			return false
		}
	}

	return true
}

func extractProject(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
