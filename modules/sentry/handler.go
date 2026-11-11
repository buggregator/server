package sentry

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"

	"github.com/buggregator/server/v2/internal/event"
	httpserver "github.com/buggregator/server/v2/internal/server/http"
)

type handler struct {
	db *sql.DB
	// notifyStored, if set, announces the kinds of structured rows an
	// envelope stored ("log", "metric", "trace").
	notifyStored func(project string, kinds []string)
}

// storedKinds collects the kinds of structured rows stored for one request,
// each once, in the order they came.
type storedKinds []string

func (k *storedKinds) add(kind string) {
	for _, existing := range *k {
		if existing == kind {
			return
		}
	}
	*k = append(*k, kind)
}

// announce tells the browser what was stored, when anything was.
func (h *handler) announce(project string, kinds storedKinds) {
	if h.db == nil || h.notifyStored == nil || len(kinds) == 0 {
		return
	}
	h.notifyStored(project, kinds)
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
	path := strings.TrimRight(r.URL.Path, "/")
	isSentryStore := strings.HasSuffix(path, "/store") && !strings.Contains(path, "/profiler/")
	return isSentryStore || strings.HasSuffix(path, "/envelope")
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	body, err := readCapped(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	// Sentry SDKs often send gzip or deflate compressed payloads.
	body = decompress(body, r.Header.Get("Content-Encoding"))

	// Project: the DSN path (/api/{project}/store), else what the detect step
	// found (X-Buggregator-Project, userinfo, Basic Auth). The same order the
	// pipeline uses for the events row, so the structured rows land in the
	// same project as the event card.
	project := resolveProject(extractProject(r.URL.Path))
	if project == "" {
		project = r.Header.Get(httpserver.HeaderDetectedProject)
	}

	// Try parsing as plain JSON first (legacy /store format).
	var parsed map[string]any
	if json.Unmarshal(body, &parsed) == nil {
		uuid := ""
		if id, ok := parsed["event_id"].(string); ok {
			uuid = id
		}

		body = normalizeExceptionList(body)

		// Resolve source code for browser/JS frames that only carry file+line.
		if enriched, changed := enrichSourceContext(body, nil); changed {
			body = enriched
		}

		// Store structured data if DB is available.
		h.storeJSONEvent(body, project)

		// Detect transactions (have spans + start_timestamp but no exception).
		// Transactions are stored in structured tables but should not appear
		// as empty cards in the global event feed.
		if isTransaction(parsed) {
			h.announce(project, storedKinds{"trace"})
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

	var kinds storedKinds
	defer func() { h.announce(project, kinds) }()

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
			// A kind is announced only when its rows were stored.
			if h.handleTransactionItem(item, envHeader.EventID, project) {
				kinds.add("trace")
			}

		case "spans", "span":
			// "span" is the Span v2 item SDK v4 sends for gen_ai spans.
			if h.handleSpansItem(item, project) {
				kinds.add("trace")
			}

		case "log":
			if h.handleLogItem(item, project) {
				kinds.add("log")
			}

		case "trace_metric":
			if h.handleMetricItem(item, project) {
				kinds.add("metric")
			}

		default:
			// Known non-essential item types are silently discarded.
			if !nonFeedItemTypes[item.Type] {
				slog.Debug("sentry: ignoring unknown envelope item type", "type", item.Type)
			}
		}
	}

	// Validate the primary payload is valid JSON; if not, discard it.
	if primaryPayload != nil && !json.Valid(primaryPayload) {
		primaryPayload = nil
	}

	// If no primary payload was extracted, fall back to envelope wrapping.
	// Skip structured-only item types — they are stored in their own tables.
	if primaryPayload == nil {
		primaryType = "sentry"
		for _, item := range items {
			if nonFeedItemTypes[item.Type] {
				continue
			}
			if json.Valid([]byte(item.Payload)) {
				primaryPayload = item.Payload
				break
			}
		}
	}

	// If still no payload, the envelope only contained non-feed items
	// (transactions, spans, logs, metrics). Don't create a canonical event.
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
	parsed := true
	if err := json.Unmarshal(item.Payload, &ev); err != nil {
		slog.Warn("sentry: failed to parse error event", "err", err)
		// Still show the event in the feed: only structured storage is skipped.
		parsed = false
		var idOnly struct {
			EventID string `json:"event_id"`
		}
		_ = json.Unmarshal(item.Payload, &idOnly)
		ev.EventID = idOnly.EventID
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

	payload = normalizeExceptionList(payload)

	// Resolve source code for browser/JS frames that only carry file+line.
	if enriched, changed := enrichSourceContext(payload, nil); changed {
		payload = enriched
		_ = json.Unmarshal(payload, &ev) // re-parse so structured storage carries the code too
	}

	if parsed && h.db != nil {
		if _, err := storeErrorEvent(h.db, &ev, payload, project); err != nil {
			slog.Warn("sentry: failed to store structured error event", "err", err)
		}
	}

	return uuid, payload
}

// handleTransactionItem processes a "transaction" envelope item.
// envelopeEventID is used as a fallback for SDKs that only include event_id
// in the envelope header and omit it in the transaction payload. It reports
// whether the transaction was stored.
func (h *handler) handleTransactionItem(item EnvelopeItem, envelopeEventID, project string) bool {
	var txn Transaction
	if err := json.Unmarshal(item.Payload, &txn); err != nil {
		slog.Warn("sentry: failed to parse transaction", "err", err)
		return false
	}

	payload := item.Payload
	if txn.EventID == "" && envelopeEventID != "" {
		txn.EventID = envelopeEventID
		payload = injectEventID(payload, envelopeEventID)
	}

	if h.db == nil {
		return false
	}
	if _, err := storeTransaction(h.db, &txn, payload, project); err != nil {
		slog.Warn("sentry: failed to store structured transaction", "err", err)
		return false
	}
	return true
}

// handleSpansItem processes a "spans" envelope item (Span v2 format). It
// reports whether any spans were stored.
func (h *handler) handleSpansItem(item EnvelopeItem, project string) bool {
	if h.db == nil {
		return false
	}

	// Spans v2 can be either {"items": [...]} or a bare array [...].
	var envelope SpansEnvelope
	if err := json.Unmarshal(item.Payload, &envelope); err == nil && len(envelope.Items) > 0 {
		if err := storeSpansV2(h.db, envelope.Items, nil, project); err != nil {
			slog.Warn("sentry: failed to store spans v2", "err", err)
			return false
		}
		return true
	}

	// Try as bare array.
	var spans []RawSpan
	if err := json.Unmarshal(item.Payload, &spans); err == nil && len(spans) > 0 {
		if err := storeSpansV2(h.db, spans, nil, project); err != nil {
			slog.Warn("sentry: failed to store spans v2 (array)", "err", err)
			return false
		}
		return true
	}
	return false
}

// handleLogItem processes a "log" envelope item. It reports whether any logs
// were stored.
func (h *handler) handleLogItem(item EnvelopeItem, project string) bool {
	if h.db == nil {
		return false
	}

	// Log items can be {"items": [...]} or a bare array.
	var envelope LogEnvelope
	if err := json.Unmarshal(item.Payload, &envelope); err == nil && envelope.Items != nil {
		if len(envelope.Items) == 0 {
			return false // {"items": []}: nothing to store, nothing dropped
		}
		if err := storeLogs(h.db, envelope.Items, project); err != nil {
			slog.Warn("sentry: failed to store logs", "err", err)
			return false
		}
		return true
	}

	var logs []LogRecord
	if err := json.Unmarshal(item.Payload, &logs); err != nil {
		slog.Debug("sentry: dropped log item, neither {\"items\": [...]} nor an array", "err", err)
		return false
	}
	if len(logs) == 0 {
		return false
	}
	if err := storeLogs(h.db, logs, project); err != nil {
		slog.Warn("sentry: failed to store logs (array)", "err", err)
		return false
	}
	return true
}

// handleMetricItem processes a "trace_metric" envelope item. It reports
// whether any metrics were stored.
func (h *handler) handleMetricItem(item EnvelopeItem, project string) bool {
	if h.db == nil {
		return false
	}

	// Metric items can be {"items": [...]} or a bare array.
	var envelope MetricEnvelope
	if err := json.Unmarshal(item.Payload, &envelope); err == nil && envelope.Items != nil {
		if len(envelope.Items) == 0 {
			return false // {"items": []}: nothing to store, nothing dropped
		}
		if err := storeMetrics(h.db, envelope.Items, project); err != nil {
			slog.Warn("sentry: failed to store metrics", "err", err)
			return false
		}
		return true
	}

	var metrics []MetricRecord
	if err := json.Unmarshal(item.Payload, &metrics); err != nil {
		slog.Debug("sentry: dropped metric item, neither {\"items\": [...]} nor an array", "err", err)
		return false
	}
	if len(metrics) == 0 {
		return false
	}
	if err := storeMetrics(h.db, metrics, project); err != nil {
		slog.Warn("sentry: failed to store metrics (array)", "err", err)
		return false
	}
	return true
}

// nonFeedItemTypes are envelope item types that never become a card in the
// event feed: they are stored in structured tables or discarded.
var nonFeedItemTypes = map[string]bool{
	"transaction": true, "spans": true, "span": true, "log": true, "trace_metric": true,
	"session": true, "sessions": true, "client_report": true, "check_in": true,
	"metric_buckets": true, "statsd": true,
	"profile": true, "profile_chunk": true, "user_report": true, "attachment": true,
	"replay_event": true, "replay_recording": true,
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

// normalizeExceptionList rewrites "exception": [...] (sentry-go and the
// protocol's short form) to "exception": {"values": [...]}, the form the
// frontend reads.
func normalizeExceptionList(payload json.RawMessage) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(payload, &obj); err != nil {
		return payload
	}
	exc := bytes.TrimSpace(obj["exception"])
	if len(exc) == 0 || exc[0] != '[' {
		return payload
	}
	obj["exception"] = json.RawMessage(`{"values":` + string(exc) + `}`)
	result, err := json.Marshal(obj)
	if err != nil {
		return payload
	}
	return result
}

// decompress handles gzip, deflate, zstd and br Content-Encoding.
// Also auto-detects gzip/zlib by magic bytes if header is missing.
func decompress(data []byte, encoding string) []byte {
	if len(data) == 0 {
		return data
	}

	// tooLarge: a decoder hit maxPayloadSize; the body stays as it came and
	// no other decoder tries it again.
	tooLarge := false
	ok := func(d []byte, err error) bool {
		tooLarge = tooLarge || errors.Is(err, errPayloadTooLarge)
		return err == nil
	}

	// Try by Content-Encoding header first.
	switch strings.ToLower(encoding) {
	case "gzip":
		if d, err := decompressGzip(data); ok(d, err) {
			return d
		}
	case "deflate":
		if d, err := decompressZlib(data); ok(d, err) {
			return d
		}
	case "zstd":
		if d, err := decompressZstd(data); ok(d, err) {
			return d
		}
	case "br":
		// sentry-python sends Brotli when the brotli package is installed.
		// Brotli has no magic bytes, so it is decoded only on the header.
		if d, err := readCapped(brotli.NewReader(bytes.NewReader(data))); ok(d, err) {
			return d
		}
	}

	if tooLarge {
		slog.Warn("sentry: decompressed payload over 64 MB, dropped", "encoding", encoding)
		return data
	}

	// Auto-detect by magic bytes (Sentry SDKs sometimes omit the header).
	if len(data) >= 4 {
		// Zstd magic: 0x28 0xb5 0x2f 0xfd
		if data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
			if d, err := decompressZstd(data); ok(d, err) {
				return d
			}
		}
	}
	if len(data) >= 2 {
		// Gzip magic: 0x1f 0x8b
		if data[0] == 0x1f && data[1] == 0x8b {
			if d, err := decompressGzip(data); ok(d, err) {
				return d
			}
		}
		// Zlib magic: 0x78 (0x01, 0x5e, 0x9c, 0xda)
		if data[0] == 0x78 {
			if d, err := decompressZlib(data); ok(d, err) {
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
	return readCapped(r)
}

func decompressZstd(data []byte) ([]byte, error) {
	// The decoder allocates the window a frame declares before any read:
	// cap it at the HTTP zstd limit (RFC 9659: 8 MB), else a 10-byte body
	// asks for 512 MB.
	r, err := zstd.NewReader(bytes.NewReader(data), zstd.WithDecoderMaxWindow(8<<20), zstd.WithDecoderConcurrency(1))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return readCapped(r)
}

func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return readCapped(r)
}

// maxPayloadSize caps a Sentry request body, raw and after decompression,
// so a small compressed body cannot expand into gigabytes.
const maxPayloadSize = 64 << 20

var errPayloadTooLarge = fmt.Errorf("sentry payload larger than %d MB", maxPayloadSize>>20)

// readCapped reads r whole, or fails past maxPayloadSize.
func readCapped(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxPayloadSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPayloadSize {
		return nil, errPayloadTooLarge
	}
	return data, nil
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

// resolveProject maps a Sentry DSN project segment to a Buggregator project key.
// The Sentry JavaScript SDK rejects non-numeric DSN project ids, so a purely
// numeric segment is the SDK-internal project id, not a Buggregator project name —
// it resolves to empty so the event lands in the default project. Issue #338.
func resolveProject(segment string) string {
	if isAllDigits(segment) {
		return ""
	}
	return segment
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
