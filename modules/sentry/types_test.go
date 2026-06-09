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
		"array form":   {`[{"category":"console"},{"category":"ui.click"},{"category":"navigation"}]`, 3},
		"null":         {`null`, 0},
		"empty array":  {`[]`, 0},
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
