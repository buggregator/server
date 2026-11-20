package sentry

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/buggregator/server/v2/internal/event"
	httpserver "github.com/buggregator/server/v2/internal/server/http"
)

// OTLP/HTTP receiver (issue #459): OpenTelemetry traces and logs, protobuf or
// JSON, stored in the Sentry span and log tables with source "otel", so the
// Sentry Traces and Logs pages show them. Paths:
//
//	/v1/traces, /v1/logs                                  (OTEL_EXPORTER_OTLP_ENDPOINT=http://host:8000)
//	/api/{project}/integration/otlp/v1/traces|logs        (Sentry's per-project OTLP endpoint)
//
// Only the OTLP data messages are decoded (TracesData, LogsData: the same
// wire shape as Export*ServiceRequest), never the collector service
// packages, which would pull gRPC into the binary.

const (
	sourceSentry = "sentry"
	sourceOTel   = "otel"
)

type otlpSignal int

const (
	otlpNone otlpSignal = iota
	otlpTraces
	otlpLogs
)

// otlpSignalOf tells which OTLP signal a path is for, or otlpNone.
func otlpSignalOf(path string) otlpSignal {
	p := strings.TrimRight(path, "/")
	var rest string
	switch {
	case p == "/v1/traces" || p == "/v1/logs":
		rest = p
	case strings.HasPrefix(p, "/api/"):
		_, after, ok := strings.Cut(p[len("/api/"):], "/integration/otlp")
		if !ok {
			return otlpNone
		}
		rest = after
	default:
		return otlpNone
	}
	switch rest {
	case "/v1/traces":
		return otlpTraces
	case "/v1/logs":
		return otlpLogs
	}
	return otlpNone
}

func isOTLPPath(path string) bool { return otlpSignalOf(path) != otlpNone }

// otlpProject: the path's project segment (/api/{project}/integration/otlp/…),
// else the detected project, else the X-Buggregator-Project header, else the DSN public key
// of x-sentry-auth when it names a project; empty means default.
func (h *handler) otlpProject(r *http.Request) string {
	if rest, ok := strings.CutPrefix(r.URL.Path, "/api/"); ok {
		segment, _, _ := strings.Cut(rest, "/")
		if p := resolveProject(segment); p != "" {
			return p
		}
	}
	if p := r.Header.Get(httpserver.HeaderDetectedProject); p != "" {
		return p
	}
	// OTLP exporters take extra headers (OTEL_EXPORTER_OTLP_HEADERS,
	// otlphttp headers:), so the project header alone is enough here.
	if p := strings.TrimSpace(r.Header.Get("X-Buggregator-Project")); p != "" {
		return p
	}
	return h.projectByKey(publicKey(r))
}

func isOTLPJSON(r *http.Request) bool {
	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return mt == "application/json"
}

func (h *handler) handleOTLP(r *http.Request) (*event.Incoming, error) {
	defer r.Body.Close()
	body, err := readCapped(r.Body)
	if err != nil {
		return nil, err
	}
	body = decompress(body, r.Header.Get("Content-Encoding"))

	asJSON := isOTLPJSON(r)
	project := h.otlpProject(r)

	var rejected int64
	var kind string
	switch otlpSignalOf(r.URL.Path) {
	case otlpTraces:
		var td tracepb.TracesData
		if err := decodeOTLP(body, asJSON, &td); err != nil {
			return nil, fmt.Errorf("otlp traces: %w", err)
		}
		spans, bad := otlpSpans(&td)
		rejected = bad
		if h.db != nil && len(spans) > 0 {
			if err := storeSpansV2(h.db, spans, nil, project); err != nil {
				return nil, fmt.Errorf("otlp traces: store: %w", err)
			}
			kind = "trace"
		}
	case otlpLogs:
		var ld logspb.LogsData
		if err := decodeOTLP(body, asJSON, &ld); err != nil {
			return nil, fmt.Errorf("otlp logs: %w", err)
		}
		logs, bad := otlpLogRecords(&ld)
		rejected = bad
		if h.db != nil && len(logs) > 0 {
			if err := storeLogs(h.db, logs, project); err != nil {
				return nil, fmt.Errorf("otlp logs: store: %w", err)
			}
			kind = "log"
		}
	}
	if kind != "" {
		h.announce(project, storedKinds{kind})
	}

	signal := otlpSignalOf(r.URL.Path)
	return &event.Incoming{
		Type:    "sentry",
		NoEvent: true,
		Respond: func(w http.ResponseWriter) { writeOTLPResponse(w, asJSON, signal, rejected) },
	}, nil
}

// decodeOTLP reads an OTLP message as protobuf or as OTLP/JSON.
func decodeOTLP(body []byte, asJSON bool, msg proto.Message) error {
	if !asJSON {
		return proto.Unmarshal(body, msg)
	}
	fixed, err := otlpJSONIDs(body)
	if err != nil {
		return err
	}
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(fixed, msg)
}

// otlpIDKeys are the OTLP/JSON fields that carry trace and span ids. OTLP/JSON
// writes them as hex; protojson reads bytes as base64.
var otlpIDKeys = map[string]bool{
	"traceId": true, "spanId": true, "parentSpanId": true,
	"trace_id": true, "span_id": true, "parent_span_id": true,
}

// otlpJSONIDs rewrites hex trace/span ids of an OTLP/JSON body to base64.
func otlpJSONIDs(body []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber() // int64 fields stay exact
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	var walk func(any)
	walk = func(n any) {
		switch t := n.(type) {
		case map[string]any:
			for k, val := range t {
				if s, ok := val.(string); ok && otlpIDKeys[k] {
					if b, err := hex.DecodeString(s); err == nil {
						t[k] = base64.StdEncoding.EncodeToString(b)
					}
					continue
				}
				walk(val)
			}
		case []any:
			for _, val := range t {
				walk(val)
			}
		}
	}
	walk(v)
	return json.Marshal(v)
}

// writeOTLPResponse answers with an Export*ServiceResponse: empty, or with
// partial_success when some records were rejected.
func writeOTLPResponse(w http.ResponseWriter, asJSON bool, signal otlpSignal, rejected int64) {
	msg := "some records had an invalid trace or span id"
	if asJSON {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if rejected == 0 {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		field := "rejectedSpans"
		if signal == otlpLogs {
			field = "rejectedLogRecords"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"partialSuccess": map[string]any{field: fmt.Sprint(rejected), "errorMessage": msg},
		})
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	if rejected == 0 {
		return
	}
	// partial_success = 1 { rejected_* = 1 (int64); error_message = 2 }
	var inner []byte
	inner = protowire.AppendTag(inner, 1, protowire.VarintType)
	inner = protowire.AppendVarint(inner, uint64(rejected))
	inner = protowire.AppendTag(inner, 2, protowire.BytesType)
	inner = protowire.AppendString(inner, msg)
	var out []byte
	out = protowire.AppendTag(out, 1, protowire.BytesType)
	out = protowire.AppendBytes(out, inner)
	_, _ = w.Write(out)
}

// Span flags (OTLP SpanFlags): the parent context is known to be remote.
const (
	spanFlagHasIsRemote = 0x100
	spanFlagIsRemote    = 0x200
)

// otlpSpans converts OTLP spans to Span v2 rows. A span whose trace or span
// id is not 16/8 bytes is rejected and counted.
func otlpSpans(td *tracepb.TracesData) ([]RawSpan, int64) {
	var out []RawSpan
	var rejected int64

	for _, rs := range td.GetResourceSpans() {
		resource := otlpAttrMap(rs.GetResource().GetAttributes())
		service, _ := resource["service.name"].(string)

		// Span ids of this resource, to tell a local parent from a remote one.
		local := map[string]bool{}
		for _, ss := range rs.GetScopeSpans() {
			for _, sp := range ss.GetSpans() {
				local[hex.EncodeToString(sp.GetSpanId())] = true
			}
		}

		for _, ss := range rs.GetScopeSpans() {
			scope := otlpScope(ss.GetScope())
			for _, sp := range ss.GetSpans() {
				if len(sp.GetTraceId()) != 16 || len(sp.GetSpanId()) != 8 {
					rejected++
					slog.Warn("sentry: otlp span with invalid id skipped",
						"trace_id_len", len(sp.GetTraceId()), "span_id_len", len(sp.GetSpanId()))
					continue
				}
				out = append(out, otlpSpan(sp, resource, scope, service, local))
			}
		}
	}
	return out, rejected
}

func otlpSpan(sp *tracepb.Span, resource map[string]any, scope map[string]any, service string, local map[string]bool) RawSpan {
	attrs := otlpAttrMap(sp.GetAttributes())
	kind := otlpKindName(sp.GetKind())
	parent := ""
	if len(sp.GetParentSpanId()) > 0 {
		parent = hex.EncodeToString(sp.GetParentSpanId())
	}

	// Data: flat string attributes, what the span classifier and the
	// transaction columns read.
	data := SpanData{}
	for k, v := range attrs {
		if s, err := spanDataString(v); err == nil {
			data[k] = s
		}
	}
	if service != "" {
		data["service.name"] = service
	}
	if v, ok := resource["service.version"].(string); ok && v != "" {
		data["sentry.release"] = v
	}
	for _, key := range []string{"deployment.environment.name", "deployment.environment"} {
		if v, ok := resource[key].(string); ok && v != "" {
			data["sentry.environment"] = v
			break
		}
	}

	status := "ok"
	statusCode := "unset"
	switch sp.GetStatus().GetCode() {
	case tracepb.Status_STATUS_CODE_OK:
		statusCode = "ok"
	case tracepb.Status_STATUS_CODE_ERROR:
		statusCode = "error"
		status = "internal_error"
	}

	// Stored attributes: the span's own, plus the OTel parts the UI shows.
	stored := make(map[string]any, len(attrs)+8)
	for k, v := range attrs {
		stored[k] = v
	}
	stored["otel.kind"] = kind
	stored["otel.status_code"] = statusCode
	if d := sp.GetStatus().GetMessage(); d != "" {
		stored["otel.status_description"] = d
	}
	if events := otlpEvents(sp.GetEvents()); len(events) > 0 {
		stored["otel.events"] = events
	}
	if links := otlpLinks(sp.GetLinks()); len(links) > 0 {
		stored["otel.links"] = links
	}
	if len(resource) > 0 {
		stored["otel.resource"] = resource
	}
	if scope != nil {
		stored["otel.scope"] = scope
	}

	// Segment: the service's entry span. No parent (the trace root), a parent
	// flagged remote, or a server/consumer span whose parent is not in this
	// resource's batch.
	flags := sp.GetFlags()
	segment := parent == "" ||
		(flags&spanFlagHasIsRemote != 0 && flags&spanFlagIsRemote != 0) ||
		((sp.GetKind() == tracepb.Span_SPAN_KIND_SERVER || sp.GetKind() == tracepb.Span_SPAN_KIND_CONSUMER) && !local[parent])

	return RawSpan{
		SpanID:         hex.EncodeToString(sp.GetSpanId()),
		ParentSpanID:   parent,
		TraceID:        hex.EncodeToString(sp.GetTraceId()),
		Op:             otlpOp(sp.GetKind(), attrs),
		Description:    sp.GetName(),
		Status:         status,
		StartTimestamp: unixNanoTS(sp.GetStartTimeUnixNano()),
		Timestamp:      unixNanoTS(sp.GetEndTimeUnixNano()),
		IsSegment:      segment,
		Data:           data,
		Source:         sourceOTel,
		Attrs:          stored,
	}
}

// otlpOp derives a Sentry-style op from the span kind and its semantic
// convention attributes.
func otlpOp(kind tracepb.Span_SpanKind, attrs map[string]any) string {
	has := func(keys ...string) bool {
		for _, k := range keys {
			if _, ok := attrs[k]; ok {
				return true
			}
		}
		return false
	}
	switch {
	case has("http.request.method", "http.method"):
		if kind == tracepb.Span_SPAN_KIND_CLIENT {
			return "http.client"
		}
		return "http.server"
	case has("db.system", "db.system.name"):
		return "db"
	case has("messaging.system"):
		switch kind {
		case tracepb.Span_SPAN_KIND_PRODUCER:
			return "queue.publish"
		case tracepb.Span_SPAN_KIND_CONSUMER:
			return "queue.process"
		}
		return "queue"
	case has("rpc.system"):
		return "rpc"
	}
	return otlpKindName(kind)
}

func otlpKindName(kind tracepb.Span_SpanKind) string {
	switch kind {
	case tracepb.Span_SPAN_KIND_SERVER:
		return "server"
	case tracepb.Span_SPAN_KIND_CLIENT:
		return "client"
	case tracepb.Span_SPAN_KIND_PRODUCER:
		return "producer"
	case tracepb.Span_SPAN_KIND_CONSUMER:
		return "consumer"
	}
	return "internal"
}

func otlpEvents(events []*tracepb.Span_Event) []map[string]any {
	var out []map[string]any
	for _, e := range events {
		item := map[string]any{"name": e.GetName()}
		if ts := unixNanoTS(e.GetTimeUnixNano()); ts.Number() != "" {
			if t := parseTimestamp(ts.Number()); t != nil {
				item["time"] = *t
			}
		}
		if attrs := otlpAttrMap(e.GetAttributes()); len(attrs) > 0 {
			item["attributes"] = attrs
		}
		out = append(out, item)
	}
	return out
}

func otlpLinks(links []*tracepb.Span_Link) []map[string]any {
	var out []map[string]any
	for _, l := range links {
		item := map[string]any{
			"trace_id": hex.EncodeToString(l.GetTraceId()),
			"span_id":  hex.EncodeToString(l.GetSpanId()),
		}
		if attrs := otlpAttrMap(l.GetAttributes()); len(attrs) > 0 {
			item["attributes"] = attrs
		}
		out = append(out, item)
	}
	return out
}

func otlpScope(s *commonpb.InstrumentationScope) map[string]any {
	if s == nil || (s.GetName() == "" && s.GetVersion() == "") {
		return nil
	}
	out := map[string]any{"name": s.GetName()}
	if s.GetVersion() != "" {
		out["version"] = s.GetVersion()
	}
	return out
}

// otlpLogRecords converts OTLP log records to Sentry log rows. A record with
// a trace or span id of the wrong length is rejected and counted.
func otlpLogRecords(ld *logspb.LogsData) ([]LogRecord, int64) {
	var out []LogRecord
	var rejected int64
	for _, rl := range ld.GetResourceLogs() {
		resource := otlpAttrMap(rl.GetResource().GetAttributes())
		service, _ := resource["service.name"].(string)
		for _, sl := range rl.GetScopeLogs() {
			scope := otlpScope(sl.GetScope())
			for _, lr := range sl.GetLogRecords() {
				if n := len(lr.GetTraceId()); n != 0 && n != 16 {
					rejected++
					slog.Warn("sentry: otlp log with invalid trace id skipped", "trace_id_len", n)
					continue
				}
				if n := len(lr.GetSpanId()); n != 0 && n != 8 {
					rejected++
					slog.Warn("sentry: otlp log with invalid span id skipped", "span_id_len", n)
					continue
				}
				out = append(out, otlpLog(lr, resource, scope, service))
			}
		}
	}
	return out, rejected
}

func otlpLog(lr *logspb.LogRecord, resource, scope map[string]any, service string) LogRecord {
	attrs := otlpAttrMap(lr.GetAttributes())
	if attrs == nil {
		attrs = map[string]any{}
	}
	if len(resource) > 0 {
		attrs["otel.resource"] = resource
	}
	if scope != nil {
		attrs["otel.scope"] = scope
	}

	ts := lr.GetTimeUnixNano()
	if ts == 0 {
		ts = lr.GetObservedTimeUnixNano()
	}

	rec := LogRecord{
		Level:          otlpLevel(lr.GetSeverityNumber(), lr.GetSeverityText()),
		SeverityNumber: int(lr.GetSeverityNumber()),
		Body:           otlpBody(lr.GetBody()),
		Timestamp:      unixNanoTS(ts),
		Attributes:     attrs,
		Source:         sourceOTel,
		ServiceName:    service,
	}
	if len(lr.GetTraceId()) == 16 {
		rec.TraceID = hex.EncodeToString(lr.GetTraceId())
	}
	if len(lr.GetSpanId()) == 8 {
		rec.SpanID = hex.EncodeToString(lr.GetSpanId())
	}
	return rec
}

// otlpLevel maps an OTel severity to the Sentry log levels: the number when
// set (OTel ranges 1–4 trace … 21–24 fatal), else the text.
func otlpLevel(num logspb.SeverityNumber, text string) string {
	switch n := int(num); {
	case n >= 21:
		return "fatal"
	case n >= 17:
		return "error"
	case n >= 13:
		return "warn"
	case n >= 9:
		return "info"
	case n >= 5:
		return "debug"
	case n >= 1:
		return "trace"
	}
	switch t := strings.ToLower(strings.TrimSpace(text)); t {
	case "":
		return "info"
	case "warning":
		return "warn"
	case "critical", "emergency", "alert":
		return "fatal"
	default:
		return t
	}
}

// otlpBody is the log body as text: a string as is, anything else as JSON.
func otlpBody(v *commonpb.AnyValue) string {
	if v == nil {
		return ""
	}
	if s, ok := v.GetValue().(*commonpb.AnyValue_StringValue); ok {
		return s.StringValue
	}
	b, err := json.Marshal(otlpValue(v))
	if err != nil {
		return ""
	}
	return string(b)
}

func otlpAttrMap(kvs []*commonpb.KeyValue) map[string]any {
	if len(kvs) == 0 {
		return nil
	}
	out := make(map[string]any, len(kvs))
	for _, kv := range kvs {
		out[kv.GetKey()] = otlpValue(kv.GetValue())
	}
	return out
}

// otlpValue turns an OTLP AnyValue into a plain Go value for JSON.
func otlpValue(v *commonpb.AnyValue) any {
	switch t := v.GetValue().(type) {
	case *commonpb.AnyValue_StringValue:
		return t.StringValue
	case *commonpb.AnyValue_BoolValue:
		return t.BoolValue
	case *commonpb.AnyValue_IntValue:
		return t.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return t.DoubleValue
	case *commonpb.AnyValue_BytesValue:
		return base64.StdEncoding.EncodeToString(t.BytesValue)
	case *commonpb.AnyValue_ArrayValue:
		out := make([]any, 0, len(t.ArrayValue.GetValues()))
		for _, item := range t.ArrayValue.GetValues() {
			out = append(out, otlpValue(item))
		}
		return out
	case *commonpb.AnyValue_KvlistValue:
		m := otlpAttrMap(t.KvlistValue.GetValues())
		if m == nil {
			m = map[string]any{}
		}
		return m
	}
	return nil
}

// unixNanoTS is epoch nanoseconds as the epoch-seconds FlexibleTS the Sentry
// storage reads ("1795020502.327121000"), exact, without float64.
func unixNanoTS(ns uint64) FlexibleTS {
	if ns == 0 {
		return FlexibleTS{}
	}
	return FlexibleTS{value: json.Number(fmt.Sprintf("%d.%09d", ns/1e9, ns%1e9))}
}
