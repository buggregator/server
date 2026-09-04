package sentry

import (
	"encoding/json"
	"testing"
)

// JS SDKs (sentry.javascript.*) send breadcrumbs as a bare array, while
// Python/PHP send the interface object {"values": [...]}. Both must parse, or
// the whole error event fails to unmarshal and never reaches the sentry tables.
func TestBreadcrumbList_AcceptsBothShapes(t *testing.T) {
	cases := map[string]struct {
		json string
		want int
	}{
		"object form": {`{"values":[{"category":"console"},{"category":"ui.click"}]}`, 2},
		"array form":  {`[{"category":"console"},{"category":"ui.click"},{"category":"navigation"}]`, 3},
		"null":        {`null`, 0},
		"empty array": {`[]`, 0},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var bl BreadcrumbList
			if err := json.Unmarshal([]byte(tc.json), &bl); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(bl.Values) != tc.want {
				t.Fatalf("got %d breadcrumbs, want %d", len(bl.Values), tc.want)
			}
		})
	}
}

// Regression: a full SvelteKit browser error event (breadcrumbs as array) must
// unmarshal into ErrorEvent without error.
func TestErrorEvent_SvelteKitBrowserPayload(t *testing.T) {
	payload := []byte(`{
		"event_id":"a8d5cb1ab96b4736918953b6468a0ee9",
		"platform":"javascript",
		"level":"error",
		"exception":{"values":[{"type":"Error","value":"sentry-test client error (browser)",
			"stacktrace":{"frames":[{"filename":"https://app/src/+page.svelte","lineno":16,"colno":10,"in_app":true}]},
			"mechanism":{"type":"auto.browser.browserapierrors.setTimeout","handled":false}}]},
		"breadcrumbs":[
			{"category":"console","level":"warning","message":"hi"},
			{"category":"ui.click","message":"button"}
		]
	}`)

	var ev ErrorEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		t.Fatalf("SvelteKit payload failed to parse: %v", err)
	}
	if ev.Breadcrumbs == nil || len(ev.Breadcrumbs.Values) != 2 {
		t.Fatalf("breadcrumbs not parsed: %+v", ev.Breadcrumbs)
	}
	if ev.Exception == nil || len(ev.Exception.Values) != 1 {
		t.Fatalf("exception not parsed: %+v", ev.Exception)
	}
}

// Nested envelope structures get ISO timestamps too, not just the top-level
// error event: PHP/Laravel SDKs put "2026-09-03T11:04:16.035Z" into breadcrumbs
// and transactions. With json.Number there, unmarshalling failed with
// `invalid number literal, trying to unmarshal "2026-..." into Number` and the
// whole event was dropped.
func TestFlexibleTS_NestedStructures(t *testing.T) {
	t.Run("breadcrumb", func(t *testing.T) {
		var bc Breadcrumb
		if err := json.Unmarshal([]byte(`{"category":"query","timestamp":"2026-09-03T11:04:16.035Z"}`), &bc); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got := bc.Timestamp.Number().String(); got == "" {
			t.Fatal("timestamp not normalized")
		}
	})

	t.Run("transaction", func(t *testing.T) {
		var txn Transaction
		payload := `{"type":"transaction","transaction":"GET /","start_timestamp":"2026-09-03T11:04:16.035Z","timestamp":"2026-09-03T11:04:17.135Z"}`
		if err := json.Unmarshal([]byte(payload), &txn); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if computeDurationMS(txn.StartTime.Number(), txn.Timestamp.Number()) == nil {
			t.Fatal("duration not computed from ISO timestamps")
		}
	})

	t.Run("log record", func(t *testing.T) {
		var log LogRecord
		if err := json.Unmarshal([]byte(`{"body":"hi","timestamp":"2026-09-03T11:04:16Z"}`), &log); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if log.Timestamp.Number().String() == "" {
			t.Fatal("timestamp not normalized")
		}
	})
}

// Span attributes are not all strings: SDKs send numbers and booleans, and with
// map[string]string the whole spans/transaction item was rejected with
// `cannot unmarshal number into Go struct field RawSpan.spans.data of type string`.
func TestSpanData_AcceptsNonStringValues(t *testing.T) {
	var span RawSpan
	payload := `{
		"span_id":"a1b2c3d4e5f60718","trace_id":"7f0c8f5c9b2a4d1e8f3b6c5a4d2e1f09",
		"op":"http.client","start_timestamp":1774960590.1,"timestamp":1774960590.9,
		"data":{
			"http.response.status_code":200,
			"http.request.redirect_count":0,
			"server.address":"api.example.com",
			"cache.hit":true,
			"sentry.sample_rate":0.25,
			"http.response.header":null,
			"custom.tags":["a","b"]
		}
	}`
	if err := json.Unmarshal([]byte(payload), &span); err != nil {
		t.Fatalf("span with non-string data failed to parse: %v", err)
	}

	want := map[string]string{
		"http.response.status_code":   "200",
		"http.request.redirect_count": "0",
		"server.address":              "api.example.com",
		"cache.hit":                   "true",
		"sentry.sample_rate":          "0.25",
		"http.response.header":        "",
		"custom.tags":                 `["a","b"]`,
	}
	for k, v := range want {
		if span.Data[k] != v {
			t.Errorf("data[%q] = %q, want %q", k, span.Data[k], v)
		}
	}
}
