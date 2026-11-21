package sentry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testdata/sdk/feedback: an error from @sentry/node 11.11.0, captureFeedback
// on it, captureFeedback alone, and a legacy user_report on the same error
// (built by @sentry/browser 7.120.4).
func TestFeedback_SDKFixtures(t *testing.T) {
	db, feed := ingestSDK(t, "feedback")

	// Only the error is a card: feedback-only envelopes used to fall into the
	// fallback and show up as raw "sentry" cards.
	if len(feed) != 1 {
		t.Fatalf("feed events = %d, want 1 (the error)", len(feed))
	}
	errorID := feed[0].UUID

	type row struct{ eventID, feedbackID, name, email, message, url, source string }
	rows, err := db.Query(`SELECT event_id, feedback_id, name, email, message, url, source FROM sentry_feedback ORDER BY rowid`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.eventID, &r.feedbackID, &r.name, &r.email, &r.message, &r.url, &r.source); err != nil {
			t.Fatal(err)
		}
		got = append(got, r)
	}
	want := []row{
		{errorID, "*", "Jane Doe", "jane@example.com", "The pay button did nothing", "https://shop.test/cart", "feedback"},
		{"", "*", "Bob", "", "Love the new dark mode", "", "feedback"},
		{errorID, "", "Jane Doe", "jane@example.com", "It broke when I clicked pay", "", "user_report"},
	}
	if len(got) != len(want) {
		t.Fatalf("feedback rows = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		w, g := want[i], got[i]
		if w.feedbackID == "*" {
			if len(g.feedbackID) != 32 {
				t.Errorf("row %d: feedback_id %q, want the feedback event's id", i, g.feedbackID)
			}
			w.feedbackID = g.feedbackID
		}
		if g != w {
			t.Errorf("row %d:\n got %+v\nwant %+v", i, g, w)
		}
	}
}

func TestFeedback_API(t *testing.T) {
	db, _ := ingestSDK(t, "feedback")
	var errorID string
	db.QueryRow(`SELECT event_id FROM sentry_error_events`).Scan(&errorID)
	mux := http.NewServeMux()
	registerAPI(mux, db, noAuth)

	get := func(query string) (total int, items []map[string]any) {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sentry/feedback"+query, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", query, w.Code, w.Body.String())
		}
		var resp struct {
			Data []map[string]any `json:"data"`
			Meta struct {
				Total int `json:"total"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		return resp.Meta.Total, resp.Data
	}

	// The id as a dashed, upper-case UUID names the same error.
	dashed := strings.ToUpper(errorID[:8] + "-" + errorID[8:12] + "-" + errorID[12:16] + "-" + errorID[16:20] + "-" + errorID[20:])
	for _, tc := range []struct {
		query string
		want  int
	}{
		{"", 3},
		{"?event_id=" + errorID, 2},
		{"?event_id=" + dashed, 2},
		{"?event_id=00000000000000000000000000000000", 0},
		{"?project=default", 3}, // the test DB has no projects table: unknown keys land in default
		{"?project=other", 0},
		{"?limit=1&page=2", 3},
	} {
		total, items := get(tc.query)
		if total != tc.want {
			t.Errorf("%q: total %d, want %d", tc.query, total, tc.want)
		}
		if tc.query == "?limit=1&page=2" && len(items) != 1 {
			t.Errorf("%q: %d items, want 1", tc.query, len(items))
		}
	}

	_, items := get("?event_id=" + errorID)
	newest := items[0] // the user_report came last
	if newest["source"] != "user_report" || newest["message"] != "It broke when I clicked pay" ||
		newest["project"] != "default" || newest["payload"] == nil || !strings.HasSuffix(newest["received_at"].(string), "Z") {
		t.Errorf("item = %+v", newest)
	}

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/sentry/counts", nil))
	var counts map[string]int
	json.Unmarshal(w.Body.Bytes(), &counts)
	if counts["feedback"] != 3 {
		t.Errorf("counts.feedback = %d, want 3", counts["feedback"])
	}
}

// A feedback item that is not JSON is dropped, not turned into a card.
func TestFeedback_BadItem(t *testing.T) {
	db := setupTestDB(t)
	h := &handler{db: db}
	body := "{}\n{\"type\":\"feedback\"}\nnot json\n"
	inc, err := h.Handle(httptest.NewRequest("POST", "/api/1/envelope/", strings.NewReader(body)))
	if err != nil {
		t.Fatal(err)
	}
	if inc != nil {
		t.Errorf("got a card: %+v", inc)
	}
	if n := countRows(t, db, "sentry_feedback", ""); n != 0 {
		t.Errorf("feedback rows = %d, want 0", n)
	}
}
