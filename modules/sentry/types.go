package sentry

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FlexibleTS handles Sentry timestamps which can be either a numeric epoch
// (float64, used by SDK v3/v4) or an ISO 8601 string (used by SDK v2).
type FlexibleTS struct {
	value json.Number // normalized to json.Number for downstream consumers
}

func (t *FlexibleTS) UnmarshalJSON(data []byte) error {
	// Try as number first (v3/v4: 1774960590.323397).
	var num json.Number
	if json.Unmarshal(data, &num) == nil {
		t.value = num
		return nil
	}

	// Try as quoted string (v2: "2026-03-31T12:44:44Z").
	var s string
	if json.Unmarshal(data, &s) == nil {
		// Convert ISO 8601 to epoch seconds for uniform downstream handling.
		if parsed, err := time.Parse(time.RFC3339, s); err == nil {
			epoch := float64(parsed.UnixNano()) / 1e9
			t.value = json.Number(strconv.FormatFloat(epoch, 'f', 6, 64))
			return nil
		}
		// Unrecognized string — store as-is.
		t.value = json.Number(s)
		return nil
	}

	return fmt.Errorf("FlexibleTS: cannot unmarshal %s", string(data))
}

// Number returns the underlying json.Number for use with parseTimestamp.
func (t FlexibleTS) Number() json.Number { return t.value }

// FlexibleMessage handles Sentry "message" which can be a plain string
// or an object {"message":"...", "params":[...], "formatted":"..."}.
type FlexibleMessage struct {
	Text      string   // plain message or the "formatted" / "message" fallback
	Template  string   // the original template (message field from object form)
	Params    []string // interpolation parameters
}

func (m *FlexibleMessage) UnmarshalJSON(data []byte) error {
	// Try plain string first (most common).
	var s string
	if json.Unmarshal(data, &s) == nil {
		m.Text = s
		return nil
	}

	// Try structured form {"message":"...", "params":[...], "formatted":"..."}.
	var obj struct {
		Message   string   `json:"message"`
		Params    []string `json:"params"`
		Formatted string   `json:"formatted"`
	}
	if json.Unmarshal(data, &obj) == nil {
		m.Template = obj.Message
		m.Params = obj.Params
		if obj.Formatted != "" {
			m.Text = obj.Formatted
		} else {
			m.Text = obj.Message
		}
		return nil
	}

	return fmt.Errorf("FlexibleMessage: cannot unmarshal %s", string(data))
}

// String returns the display text.
func (m FlexibleMessage) String() string { return m.Text }

// EnvelopeItem represents a single parsed item from a Sentry envelope.
type EnvelopeItem struct {
	Type    string          // "event", "transaction", "spans", "log", "session", etc.
	Header  json.RawMessage // raw item header JSON
	Payload json.RawMessage // raw item body JSON
}

// EnvelopeHeader is the first line of a Sentry envelope.
type EnvelopeHeader struct {
	EventID string `json:"event_id"`
	SentAt  string `json:"sent_at"`
	DSN     string `json:"dsn"`
}

// ItemHeader is the header line before each item body.
type ItemHeader struct {
	Type   string `json:"type"`
	Length int    `json:"length"`
}

// ErrorEvent represents a parsed Sentry error event.
type ErrorEvent struct {
	EventID     string          `json:"event_id"`
	Timestamp   FlexibleTS      `json:"timestamp"`
	Platform    string          `json:"platform"`
	Level       string          `json:"level"`
	Logger      string          `json:"logger"`
	Transaction string          `json:"transaction"`
	ServerName  string          `json:"server_name"`
	Environment string          `json:"environment"`
	Release     string          `json:"release"`
	Message     FlexibleMessage `json:"message"`
	LogEntry    *LogEntry       `json:"logentry"`
	Exception   *ExceptionList  `json:"exception"`
	Breadcrumbs *BreadcrumbList `json:"breadcrumbs"`
	Contexts    *Contexts       `json:"contexts"`
	SDK         *SDK            `json:"sdk"`
	Request     json.RawMessage `json:"request"`
	Modules     json.RawMessage `json:"modules"`
	Extra       json.RawMessage `json:"extra"`
	Tags        json.RawMessage `json:"tags"`
}

type LogEntry struct {
	Message string `json:"message"`
}

type ExceptionList struct {
	Values []ExceptionValue `json:"values"`
}

type ExceptionValue struct {
	Type       string          `json:"type"`
	Value      string          `json:"value"`
	Mechanism  *Mechanism      `json:"mechanism"`
	Stacktrace json.RawMessage `json:"stacktrace"`
}

type Mechanism struct {
	Type    string `json:"type"`
	Handled *bool  `json:"handled"`
}

type BreadcrumbList struct {
	Values []Breadcrumb `json:"values"`
}

type Breadcrumb struct {
	Type      string          `json:"type"`
	Category  string          `json:"category"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Timestamp json.Number     `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

type Contexts struct {
	Trace *TraceContext `json:"trace"`
}

type TraceContext struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

type SDK struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Transaction represents a Sentry transaction envelope item.
type Transaction struct {
	EventID      string          `json:"event_id"`
	Type         string          `json:"type"`
	Transaction  string          `json:"transaction"`
	Timestamp    json.Number     `json:"timestamp"`
	StartTime    json.Number     `json:"start_timestamp"`
	Platform     string          `json:"platform"`
	Environment  string          `json:"environment"`
	Release      string          `json:"release"`
	Contexts     *Contexts       `json:"contexts"`
	Spans        []RawSpan       `json:"spans"`
	Measurements json.RawMessage `json:"measurements"`
	SDK          *SDK            `json:"sdk"`

	// Root span fields
	Op     string `json:"op,omitempty"`
	Status string `json:"status,omitempty"`
}

// RawSpan represents a span within a transaction or spans envelope item.
type RawSpan struct {
	SpanID         string            `json:"span_id"`
	ParentSpanID   string            `json:"parent_span_id"`
	TraceID        string            `json:"trace_id"`
	Op             string            `json:"op"`
	Description    string            `json:"description"`
	Status         string            `json:"status"`
	StartTimestamp json.Number       `json:"start_timestamp"`
	Timestamp      json.Number       `json:"timestamp"`
	IsSegment      bool              `json:"is_segment"`
	Data           map[string]string `json:"data"`
}

// SpansEnvelope represents a Sentry spans (v2) envelope item body.
type SpansEnvelope struct {
	Items []RawSpan `json:"items"`
}

// LogEnvelope represents a Sentry log envelope item body.
type LogEnvelope struct {
	Items []LogRecord `json:"items"`
}

// LogRecord represents a single Sentry native log entry.
type LogRecord struct {
	TraceID        string            `json:"trace_id"`
	SpanID         string            `json:"span_id"`
	Level          string            `json:"level"`
	SeverityNumber int               `json:"severity_number"`
	Body           string            `json:"body"`
	Timestamp      json.Number       `json:"timestamp"`
	Attributes     map[string]any    `json:"attributes"`
}

// effectiveMessage returns the event message, checking logentry fallback.
func (e *ErrorEvent) effectiveMessage() string {
	if e.Message.Text != "" {
		return e.Message.Text
	}
	if e.LogEntry != nil {
		return e.LogEntry.Message
	}
	return ""
}

// traceID extracts trace_id from contexts.trace if available.
func (e *ErrorEvent) traceID() string {
	if e.Contexts != nil && e.Contexts.Trace != nil {
		return e.Contexts.Trace.TraceID
	}
	return ""
}

// spanID extracts span_id from contexts.trace if available.
func (e *ErrorEvent) spanID() string {
	if e.Contexts != nil && e.Contexts.Trace != nil {
		return e.Contexts.Trace.SpanID
	}
	return ""
}
