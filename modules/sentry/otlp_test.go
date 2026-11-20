package sentry

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Fixtures in testdata/otlp (issue #459):
//   - otelgo_traces.pb.gz, otelgo_logs.pb: what the otel-go SDK v1.47.0
//     (otlptracehttp gzip, otlploghttp) sends for one checkout-api request:
//     a server span with a remote parent, a failing db span with an
//     exception event, an http client span with a link; two log records.
//   - collector_traces.json, collector_logs.json: the same data after
//     otel/opentelemetry-collector-contrib:0.145.0, exporter otlphttp with
//     encoding: json (hex ids, int64 as strings, kind as numbers).

const otlpTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"

func otlpRequest(t *testing.T, path, contentType, encoding, fixture string) *http.Request {
	t.Helper()
	body, err := os.ReadFile("testdata/otlp/" + fixture)
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	r.Header.Set("Content-Type", contentType)
	if encoding != "" {
		r.Header.Set("Content-Encoding", encoding)
	}
	return r
}

// serveOTLP runs a request through the handler and writes its answer.
func serveOTLP(t *testing.T, h *handler, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	if !h.Match(r) {
		t.Fatalf("handler does not match %s", r.URL.Path)
	}
	inc, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if inc == nil || !inc.NoEvent || inc.Respond == nil {
		t.Fatalf("OTLP must answer without a feed event, got %+v", inc)
	}
	rec := httptest.NewRecorder()
	inc.Respond(rec)
	return rec
}

func apiGet(t *testing.T, db *sql.DB, url string) (int, map[string]any) {
	t.Helper()
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body
}

func listData(t *testing.T, body map[string]any) []map[string]any {
	t.Helper()
	raw, _ := body["data"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, item.(map[string]any))
	}
	return out
}

func TestOTLP_Traces(t *testing.T) {
	tests := []struct {
		name, path, contentType, encoding, fixture string
		wantProject, wantRespType                  string
	}{
		{"otel-go SDK, protobuf + gzip, plain /v1/traces", "/v1/traces", "application/x-protobuf", "gzip", "otelgo_traces.pb.gz", "default", "application/x-protobuf"},
		{"collector otlphttp, JSON, Sentry-style project path", "/api/shop/integration/otlp/v1/traces/", "application/json", "", "collector_traces.json", "shop", "application/json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProjectsDB(t, "shop")
			var announced []string
			h := &handler{db: db, notifyStored: func(project string, kinds []string) {
				announced = append(announced, project+":"+strings.Join(kinds, ","))
			}}

			rec := serveOTLP(t, h, otlpRequest(t, tt.path, tt.contentType, tt.encoding, tt.fixture))
			if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != tt.wantRespType {
				t.Fatalf("answer %d %q, want 200 %q", rec.Code, rec.Header().Get("Content-Type"), tt.wantRespType)
			}
			// The module maps an empty project to "default" when it broadcasts.
			if want := []string{strings.TrimPrefix(tt.wantProject, "default") + ":trace"}; !reflect.DeepEqual(announced, want) {
				t.Errorf("announced %v, want %v", announced, want)
			}

			// The server span (remote parent) is the transaction.
			var name, op, status, env, release, source, service, project string
			var dur sql.NullInt64
			err := db.QueryRow(`SELECT transaction_name, op, status, environment, release, source, service_name, project_id, duration_ms
				FROM sentry_transactions WHERE trace_id = ?`, otlpTraceID).
				Scan(&name, &op, &status, &env, &release, &source, &service, &project, &dur)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{name, op, status, env, release, source, service, project}
			want := []string{"POST /checkout", "http.server", "internal_error", "local", "1.4.2", "otel", "checkout-api", tt.wantProject}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("transaction %v, want %v", got, want)
			}
			if !dur.Valid || dur.Int64 < 4 {
				t.Errorf("transaction duration_ms %v, want ≥ 4", dur)
			}

			// The two children are spans.
			rows, err := db.Query(`SELECT description, op, is_error, source, service_name FROM sentry_spans WHERE trace_id = ? ORDER BY start_ts`, otlpTraceID)
			if err != nil {
				t.Fatal(err)
			}
			var spans []string
			for rows.Next() {
				var d, o, s, svc string
				var isErr int
				if err := rows.Scan(&d, &o, &isErr, &s, &svc); err != nil {
					t.Fatal(err)
				}
				spans = append(spans, strings.Join([]string{d, o, s, svc}, "|")+map[int]string{0: "", 1: "|error"}[isErr])
			}
			rows.Close()
			wantSpans := []string{"SELECT orders|db|otel|checkout-api|error", "POST|http.client|otel|checkout-api"}
			if !reflect.DeepEqual(spans, wantSpans) {
				t.Errorf("spans %v, want %v", spans, wantSpans)
			}

			// List API: sources, services, source filter.
			code, body := apiGet(t, db, "/api/sentry/traces")
			items := listData(t, body)
			if code != 200 || len(items) != 1 {
				t.Fatalf("GET /api/sentry/traces: %d, %d items", code, len(items))
			}
			if !reflect.DeepEqual(items[0]["sources"], []any{"otel"}) || !reflect.DeepEqual(items[0]["services"], []any{"checkout-api"}) {
				t.Errorf("list item sources %v services %v", items[0]["sources"], items[0]["services"])
			}
			for filter, n := range map[string]int{"otel": 1, "sentry": 0} {
				_, body := apiGet(t, db, "/api/sentry/traces?source="+filter)
				if got := len(listData(t, body)); got != n {
					t.Errorf("?source=%s: %d traces, want %d", filter, got, n)
				}
			}
			if code, _ := apiGet(t, db, "/api/sentry/traces?source=jaeger"); code != http.StatusBadRequest {
				t.Errorf("?source=jaeger: %d, want 400", code)
			}

			// Detail API: OTel parts in span attributes.
			code, body = apiGet(t, db, "/api/sentry/traces/"+otlpTraceID)
			if code != 200 {
				t.Fatalf("trace detail: %d", code)
			}
			if txn := body["transaction"].(map[string]any); !reflect.DeepEqual(txn["sources"], []any{"otel"}) {
				t.Errorf("transaction sources %v", txn["sources"])
			}
			var dbSpan, clientSpan map[string]any
			for _, s := range body["spans"].([]any) {
				s := s.(map[string]any)
				switch s["description"] {
				case "SELECT orders":
					dbSpan = s
				case "POST":
					clientSpan = s
				}
			}
			if dbSpan == nil || clientSpan == nil {
				t.Fatalf("detail spans missing: %v", body["spans"])
			}
			if dbSpan["source"] != "otel" || dbSpan["service_name"] != "checkout-api" {
				t.Errorf("db span source %v service %v", dbSpan["source"], dbSpan["service_name"])
			}
			attrs := dbSpan["attributes"].(map[string]any)
			if attrs["otel.kind"] != "client" || attrs["otel.status_code"] != "error" || attrs["otel.status_description"] != "connection reset by peer" || attrs["db.system"] != "postgresql" {
				t.Errorf("db span attributes %v", attrs)
			}
			if tags, _ := attrs["db.tags"].([]any); len(tags) != 2 {
				t.Errorf("array attribute kept as JSON array, got %v", attrs["db.tags"])
			}
			events, _ := attrs["otel.events"].([]any)
			if len(events) != 1 {
				t.Fatalf("otel.events %v", attrs["otel.events"])
			}
			ev := events[0].(map[string]any)
			evAttrs := ev["attributes"].(map[string]any)
			if ev["name"] != "exception" || evAttrs["exception.message"] != "connection reset by peer" || evAttrs["exception.stacktrace"] == nil || ev["time"] == nil {
				t.Errorf("exception event %v", ev)
			}
			if res := attrs["otel.resource"].(map[string]any); res["service.version"] != "1.4.2" {
				t.Errorf("otel.resource %v", res)
			}
			if scope := attrs["otel.scope"].(map[string]any); scope["name"] != "github.com/acme/checkout" {
				t.Errorf("otel.scope %v", scope)
			}
			links, _ := clientSpan["attributes"].(map[string]any)["otel.links"].([]any)
			if len(links) != 1 || links[0].(map[string]any)["span_id"] != "00f067aa0ba902b7" {
				t.Errorf("otel.links %v", links)
			}
		})
	}
}

func TestOTLP_Logs(t *testing.T) {
	tests := []struct {
		name, path, contentType, fixture string
	}{
		{"otel-go SDK, protobuf", "/v1/logs", "application/x-protobuf", "otelgo_logs.pb"},
		{"collector otlphttp, JSON", "/api/1/integration/otlp/v1/logs/", "application/json; charset=utf-8", "collector_logs.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB(t)
			h := &handler{db: db}
			if rec := serveOTLP(t, h, otlpRequest(t, tt.path, tt.contentType, "", tt.fixture)); rec.Code != 200 {
				t.Fatalf("answer %d", rec.Code)
			}

			code, body := apiGet(t, db, "/api/sentry/logs?source=otel")
			logs := listData(t, body)
			if code != 200 || len(logs) != 2 {
				t.Fatalf("GET /api/sentry/logs?source=otel: %d, %d logs", code, len(logs))
			}
			byLevel := map[string]map[string]any{}
			for _, l := range logs {
				byLevel[l["level"].(string)] = l
				if l["source"] != "otel" || l["service_name"] != "checkout-api" {
					t.Errorf("log source %v service %v", l["source"], l["service_name"])
				}
			}
			e, w := byLevel["error"], byLevel["warn"]
			if e == nil || w == nil {
				t.Fatalf("levels %v", byLevel)
			}
			if e["body"] != "order lookup failed" || e["trace_id"] != otlpTraceID || e["span_id"] == nil || e["severity_number"] != float64(17) {
				t.Errorf("error log %v", e)
			}
			attrs := e["attributes"].(map[string]any)
			if attrs["order.id"] != "42" || attrs["attempt"] != float64(3) || attrs["otel.resource"] == nil || attrs["otel.scope"] == nil {
				t.Errorf("error log attributes %v", attrs)
			}
			// A structured body is kept as JSON; no time → observed time.
			if !strings.Contains(w["body"].(string), `"event":"retry"`) || w["log_ts"] == nil {
				t.Errorf("warn log %v", w)
			}

			if _, body := apiGet(t, db, "/api/sentry/logs?source=sentry"); len(listData(t, body)) != 0 {
				t.Error("?source=sentry returns OTel logs")
			}
			if code, _ := apiGet(t, db, "/api/sentry/logs?source=x"); code != http.StatusBadRequest {
				t.Errorf("?source=x: %d, want 400", code)
			}
		})
	}
}

func TestOTLP_PartialSuccess(t *testing.T) {
	good := `{"traceId":"` + otlpTraceID + `","spanId":"1111111111111111","name":"ok","kind":2,"startTimeUnixNano":"1795195095949113701","endTimeUnixNano":"1795195095952323846"}`
	bad := `{"traceId":"abcd","spanId":"2222222222222222","name":"bad"}`
	body := `{"resourceSpans":[{"scopeSpans":[{"spans":[` + bad + `,` + good + `]}]}]}`

	for _, asJSON := range []bool{true, false} {
		db := setupTestDB(t)
		h := &handler{db: db}
		r := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		if !asJSON {
			// Same data as protobuf.
			var td = mustTracesFromJSON(t, body)
			r = httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(td))
			r.Header.Set("Content-Type", "application/x-protobuf")
		}
		rec := serveOTLP(t, h, r)

		var n int
		db.QueryRow(`SELECT COUNT(*) FROM sentry_transactions`).Scan(&n)
		if n != 1 {
			t.Errorf("json=%v: %d stored, want the good span only", asJSON, n)
		}
		if asJSON {
			want := `{"partialSuccess":{"errorMessage":"some records had an invalid trace or span id","rejectedSpans":"1"}}`
			if got := strings.TrimSpace(rec.Body.String()); got != want {
				t.Errorf("JSON answer %s, want %s", got, want)
			}
		} else {
			// partial_success{rejected_spans: 1, error_message: …}
			b := rec.Body.Bytes()
			if len(b) < 4 || b[0] != 0x0a || b[2] != 0x08 || b[3] != 0x01 || !bytes.Contains(b, []byte("invalid trace or span id")) {
				t.Errorf("protobuf answer % x", b)
			}
		}
	}
}

func mustTracesFromJSON(t *testing.T, body string) []byte {
	t.Helper()
	td, err := otlpTracesFromJSONForTest([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return td
}

func TestOTLP_BadBody(t *testing.T) {
	h := &handler{db: setupTestDB(t)}
	for _, ct := range []string{"application/x-protobuf", "application/json"} {
		r := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader("\xff\xff not otlp"))
		r.Header.Set("Content-Type", ct)
		if _, err := h.Handle(r); err == nil {
			t.Errorf("%s: garbage accepted", ct)
		}
	}
}

// The OTel Collector sentry exporter names the project only by its DSN key,
// in a lower-case "sentry sentry_key=…" header.
func TestOTLP_ProjectByKey(t *testing.T) {
	db := setupProjectsDB(t, "billing")
	h := &handler{db: db}
	r := otlpRequest(t, "/api/7/integration/otlp/v1/traces/", "application/x-protobuf", "gzip", "otelgo_traces.pb.gz")
	r.Header.Set("x-sentry-auth", "sentry sentry_key=billing")
	serveOTLP(t, h, r)

	var project string
	if err := db.QueryRow(`SELECT project_id FROM sentry_transactions`).Scan(&project); err != nil {
		t.Fatal(err)
	}
	if project != "billing" {
		t.Errorf("project %q, want billing", project)
	}
}

// OTEL_EXPORTER_OTLP_HEADERS=X-Buggregator-Project=billing names the project.
// An unknown project name (path or header) lands in default, as an unknown
// DSN key does (#430).
func TestOTLP_ProjectHeader(t *testing.T) {
	tests := []struct{ name, path, header, want string }{
		{"known header", "/v1/logs", "billing", "billing"},
		{"unknown header", "/v1/logs", "nope", "default"},
		{"known path", "/api/billing/integration/otlp/v1/logs", "", "billing"},
		{"unknown path", "/api/nope/integration/otlp/v1/logs", "", "default"},
		{"unknown path, known header", "/api/nope/integration/otlp/v1/logs", "billing", "billing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupProjectsDB(t, "billing")
			h := &handler{db: db}
			r := otlpRequest(t, tt.path, "application/x-protobuf", "", "otelgo_logs.pb")
			if tt.header != "" {
				r.Header.Set("X-Buggregator-Project", tt.header)
			}
			serveOTLP(t, h, r)

			var n int
			db.QueryRow(`SELECT COUNT(*) FROM sentry_logs WHERE project_id = ?`, tt.want).Scan(&n)
			if n != 2 {
				t.Errorf("%d logs in project %s, want 2", n, tt.want)
			}
		})
	}
}

// A trace that holds both Sentry SDK spans and OTel spans lists both sources
// and matches either filter.
func TestOTLP_MixedTraceSources(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}
	sentrySpans := []RawSpan{{
		TraceID: otlpTraceID, SpanID: "00f067aa0ba902b7", Op: "pageload", Description: "/checkout", IsSegment: true,
		StartTimestamp: epochTS("1795195095.900"), Timestamp: epochTS("1795195096.000"),
		Data: SpanData{"service.name": "shop-web"},
	}}
	if err := storeSpansV2(db, sentrySpans, nil, ""); err != nil {
		t.Fatal(err)
	}
	serveOTLP(t, h, otlpRequest(t, "/v1/traces", "application/x-protobuf", "gzip", "otelgo_traces.pb.gz"))

	// One row per trace, named by its root (the Sentry pageload, no parent).
	_, body := apiGet(t, db, "/api/sentry/traces")
	items := listData(t, body)
	if len(items) != 1 {
		t.Fatalf("%d rows, want 1 per trace", len(items))
	}
	it := items[0]
	if it["transaction_name"] != "/checkout" || !reflect.DeepEqual(it["sources"], []any{"sentry", "otel"}) || !reflect.DeepEqual(it["services"], []any{"shop-web", "checkout-api"}) {
		t.Errorf("row %v sources %v services %v", it["transaction_name"], it["sources"], it["services"])
	}
	for _, f := range []string{"sentry", "otel"} {
		if _, body := apiGet(t, db, "/api/sentry/traces?source="+f); len(listData(t, body)) != 1 {
			t.Errorf("?source=%s does not match the mixed trace", f)
		}
	}
	if _, body := apiGet(t, db, "/api/sentry/counts"); body["traces"] != float64(1) {
		t.Errorf("counts traces %v, want 1", body["traces"])
	}

	// Detail: every service's entry span is in spans, so they form one tree:
	// Sentry root → checkout-api server span → its children.
	_, body = apiGet(t, db, "/api/sentry/traces/"+otlpTraceID)
	if txn := body["transaction"].(map[string]any); txn["transaction_name"] != "/checkout" || txn["span_id"] != "00f067aa0ba902b7" {
		t.Errorf("detail transaction %v span %v, want the root", txn["transaction_name"], txn["span_id"])
	}
	parents := map[string]any{}
	byID := map[string]map[string]any{}
	for _, s := range body["spans"].([]any) {
		s := s.(map[string]any)
		parents[s["description"].(string)] = s["parent_span_id"]
		byID[s["span_id"].(string)] = s
	}
	if _, listed := parents["/checkout"]; listed {
		t.Error("the root is the header, not a span")
	}
	server := byID[fmt.Sprint(parents["SELECT orders"])]
	if server == nil || server["description"] != "POST /checkout" || server["parent_span_id"] != "00f067aa0ba902b7" ||
		server["source"] != "otel" || server["service_name"] != "checkout-api" || server["is_error"] != true {
		t.Errorf("OTel entry span %v", server)
	}
	if attrs, _ := server["attributes"].(map[string]any); attrs["otel.kind"] != "server" {
		t.Errorf("OTel entry span attributes %v", server["attributes"])
	}
}

func TestOTLPSignalOf(t *testing.T) {
	tests := map[string]otlpSignal{
		"/v1/traces":                              otlpTraces,
		"/v1/traces/":                             otlpTraces,
		"/v1/logs":                                otlpLogs,
		"/api/1/integration/otlp/v1/traces/":      otlpTraces,
		"/api/shop/integration/otlp/v1/logs":      otlpLogs,
		"/v1/metrics":                             otlpNone,
		"/api/1/integration/otlp/v1/metrics/":     otlpNone,
		"/api/1/envelope/":                        otlpNone,
		"/api/1/store/":                           otlpNone,
		"/x/v1/traces":                            otlpNone,
		"/api/1/integration/otlp/v1/traces/extra": otlpNone,
	}
	for path, want := range tests {
		if got := otlpSignalOf(path); got != want {
			t.Errorf("otlpSignalOf(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestOTLPLevel(t *testing.T) {
	tests := []struct {
		num  int32
		text string
		want string
	}{
		{1, "", "trace"}, {5, "", "debug"}, {9, "", "info"}, {13, "", "warn"},
		{17, "", "error"}, {21, "", "fatal"}, {0, "Warning", "warn"}, {0, "", "info"},
		{0, "CRITICAL", "fatal"}, {0, "notice", "notice"}, {9, "ERROR", "info"},
	}
	for _, tt := range tests {
		if got := otlpLevel(otlpSeverity(tt.num), tt.text); got != tt.want {
			t.Errorf("otlpLevel(%d, %q) = %q, want %q", tt.num, tt.text, got, tt.want)
		}
	}
}

// Sentry data names its service like OTel does, so a mixed trace reads
// shop-web → checkout-api, not sentry.php → checkout-api.
func TestSentryServiceNames(t *testing.T) {
	txns := []struct {
		name string
		txn  Transaction
		want string
	}{
		{"trace data service.name", Transaction{Contexts: &Contexts{Trace: &TraceContext{Data: map[string]any{"service.name": "api"}}}, ServerName: "web-1", SDK: &SDK{Name: "sentry.php"}}, "api"},
		{"server_name", Transaction{ServerName: "shop-web", SDK: &SDK{Name: "sentry.php"}}, "shop-web"},
		{"SDK name last", Transaction{SDK: &SDK{Name: "sentry.php"}}, "sentry.php"},
		{"nothing", Transaction{}, ""},
	}
	for _, tt := range txns {
		if got := transactionService(&tt.txn); got != tt.want {
			t.Errorf("transactionService(%s) = %q, want %q", tt.name, got, tt.want)
		}
	}

	logs := []struct {
		attrs map[string]any
		want  string
	}{
		{map[string]any{"server.address": map[string]any{"value": "shop-web", "type": "string"}}, "shop-web"},
		{map[string]any{"sentry.server.address": map[string]any{"value": "go-01", "type": "string"}}, "go-01"},
		{map[string]any{"service.name": "api", "server.address": "web-1"}, "api"},
		{map[string]any{"sentry.sdk.name": "sentry.php"}, ""},
	}
	for _, tt := range logs {
		if got := sentryLogService(tt.attrs); got != tt.want {
			t.Errorf("sentryLogService(%v) = %q, want %q", tt.attrs, got, tt.want)
		}
	}

	// Stored: a PHP transaction with server_name, its spans, and a log.
	db := setupTestDB(t)
	h := &handler{db: db}
	body := `{"event_id":"e1"}
{"type":"transaction"}
{"event_id":"e1","type":"transaction","transaction":"POST /cart","server_name":"shop-web","sdk":{"name":"sentry.php"},"start_timestamp":1795195095.9,"timestamp":1795195096.0,"contexts":{"trace":{"trace_id":"` + otlpTraceID + `","span_id":"00f067aa0ba902b7","op":"http.server"}},"spans":[{"span_id":"3333333333333333","parent_span_id":"00f067aa0ba902b7","trace_id":"` + otlpTraceID + `","op":"db","start_timestamp":1795195095.91,"timestamp":1795195095.95}]}
{"type":"log"}
{"items":[{"level":"info","body":"hi","timestamp":1795195095.92,"attributes":{"server.address":{"value":"shop-web","type":"string"}}}]}
`
	if _, err := h.handleEnvelope([]byte(body), ""); err != nil {
		t.Fatal(err)
	}
	var txnSvc, spanSvc, logSvc string
	db.QueryRow(`SELECT service_name FROM sentry_transactions`).Scan(&txnSvc)
	db.QueryRow(`SELECT service_name FROM sentry_spans`).Scan(&spanSvc)
	db.QueryRow(`SELECT service_name FROM sentry_logs`).Scan(&logSvc)
	if txnSvc != "shop-web" || spanSvc != "shop-web" || logSvc != "shop-web" {
		t.Errorf("service names: transaction %q span %q log %q, want shop-web", txnSvc, spanSvc, logSvc)
	}
}
