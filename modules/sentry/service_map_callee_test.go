package sentry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

// An outgoing HTTP call is drawn to the service that received it, found by
// the callee naming the call's span as its parent: a transaction
// (contexts.trace.parent_span_id) or a span v2 (parent_span_id). A call
// nobody reports, and databases, keep their address.
func TestServiceMapCallToKnownService(t *testing.T) {
	db := setupTestDB(t)
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)
	noon := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return noon }
	t.Cleanup(func() { now = time.Now })

	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	span := func(id, txn, parent, service, peer, peerType string) {
		var txnID any
		if txn != "" {
			txnID = txn
		}
		var p any
		if peer != "" {
			p = peer
		}
		exec(`INSERT INTO sentry_spans (id, span_id, transaction_id, trace_id, parent_span_id, service_name, peer_address, peer_type, start_ts)
			VALUES (?, ?, ?, 'tr', ?, ?, ?, ?, '2026-11-16T11:30:00Z')`, id, id, txnID, parent, service, p, peerType)
	}
	exec(`INSERT INTO sentry_traces (trace_id) VALUES ('tr')`)

	// storefront → shop-api.test: shop-api's transaction names the call as parent.
	span("sf-call", "", "sf-root", "storefront", "shop-api.test", "http")
	exec(`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, start_ts, payload)
		VALUES ('shop-txn', 'e1', 'tr', 'POST /api/checkout', '2026-11-16T11:30:00Z',
			'{"contexts":{"trace":{"span_id":"shop-root","parent_span_id":"sf-call"}}}')`)
	span("shop-sql", "shop-txn", "shop-root", "shop-api", "mysql:3306", "db")
	// A child of the call in storefront itself (a retry) is not the callee.
	span("sf-retry", "", "sf-call", "storefront", "", "")
	// storefront → search.test: only a storefront child, nobody else reports it.
	span("sf-search", "", "sf-root", "storefront", "search.test", "http")
	span("sf-search-child", "", "sf-search", "storefront", "", "")
	// shop-api → billing-api:9000: billing-api sends span v2 with the call as parent.
	span("shop-call", "shop-txn", "shop-root", "shop-api", "billing-api:9000", "http")
	span("bill-server", "", "shop-call", "billing-api", "", "")
	// billing-api → api.stripe.com: nobody reports it.
	span("bill-call", "", "bill-server", "billing-api", "api.stripe.com", "http")

	// shop-api → payments-worker (OTel): the callee's entry span is a
	// transaction row with its own parent_span_id and service_name, no spans.
	span("shop-pay", "shop-txn", "shop-root", "shop-api", "payments.svc", "http")
	exec(`INSERT INTO sentry_transactions (id, event_id, trace_id, transaction_name, start_ts, payload, parent_span_id, service_name, source)
		VALUES ('pay-txn', 'e2', 'tr', 'process order', '2026-11-16T11:30:00Z', '{}', 'shop-pay', 'payments-worker', 'otel')`)

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sentry/service-map?window=60", nil))
	if w.Code != 200 {
		t.Fatalf("status = %d: %s", w.Code, w.Body)
	}
	var resp struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
		} `json:"edges"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	var edges, nodes []string
	for _, e := range resp.Edges {
		edges = append(edges, e.Source+"->"+e.Target)
	}
	for _, n := range resp.Nodes {
		nodes = append(nodes, n.ID)
	}
	sort.Strings(edges)
	sort.Strings(nodes)
	if got, want := strings.Join(edges, " "), "billing-api->api.stripe.com shop-api->billing-api shop-api->mysql:3306 shop-api->payments-worker storefront->search.test storefront->shop-api"; got != want {
		t.Errorf("edges = %s\nwant    %s", got, want)
	}
	if got, want := strings.Join(nodes, " "), "api.stripe.com billing-api mysql:3306 payments-worker search.test shop-api storefront"; got != want {
		t.Errorf("nodes = %s\nwant    %s", got, want)
	}
}
