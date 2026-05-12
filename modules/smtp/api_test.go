package smtp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/storage"
)

func setupAPITest(t *testing.T) (*http.ServeMux, *storage.SQLiteStore, *Module) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	store := storage.NewSQLiteStore(db)
	mod := New(":1025", nil, nil)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux, store)

	return mux, store, mod
}

func storeSMTPEvent(t *testing.T, store *storage.SQLiteStore, uuid string, email ParsedEmail, ts float64, project string) {
	t.Helper()
	payload, _ := json.Marshal(email)
	ev := event.Event{
		UUID:      uuid,
		Type:      "smtp",
		Payload:   payload,
		Timestamp: ts,
		Project:   project,
	}
	if err := store.Store(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
}

// TestSMTPAPI_Cursor verifies the cursor endpoint returns a valid RFC3339 time.
func TestSMTPAPI_Cursor(t *testing.T) {
	mux, _, _ := setupAPITest(t)

	r := httptest.NewRequest("GET", "/api/smtp/cursor", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if _, err := time.Parse(time.RFC3339Nano, resp["cursor"]); err != nil {
		t.Errorf("cursor %q is not valid RFC3339Nano: %v", resp["cursor"], err)
	}
}

// TestSMTPAPI_Messages_Empty verifies an empty list when no messages exist.
func TestSMTPAPI_Messages_Empty(t *testing.T) {
	mux, _, _ := setupAPITest(t)

	r := httptest.NewRequest("GET", "/api/smtp/messages", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	if len(data) != 0 {
		t.Errorf("expected empty, got %d", len(data))
	}
}

// TestSMTPAPI_Messages_Filter verifies all filter parameters.
func TestSMTPAPI_Messages_Filter(t *testing.T) {
	mux, store, _ := setupAPITest(t)

	storeSMTPEvent(t, store, "f-uuid1", ParsedEmail{
		Subject: "Hello",
		To:      []EmailAddress{{Email: "alice@example.com"}},
		From:    []EmailAddress{{Email: "sender@example.com"}},
		Text:    "Click here: https://example.com/reset/abc",
	}, 1000.0, "default")

	storeSMTPEvent(t, store, "f-uuid2", ParsedEmail{
		Subject: "World",
		To:      []EmailAddress{{Email: "bob@example.com"}},
		From:    []EmailAddress{{Email: "sender@example.com"}},
		Text:    "Your code is 123456",
	}, 1001.0, "default")

	storeSMTPEvent(t, store, "f-uuid3", ParsedEmail{
		Subject: "Other project",
		To:      []EmailAddress{{Email: "alice@example.com"}},
	}, 1002.0, "other")

	t.Run("filter by to", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?to=alice", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 2 { // alice appears in default and other project
			t.Errorf("expected 2, got %d", len(data))
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?project=default", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 2 {
			t.Errorf("expected 2, got %d", len(data))
		}
	})

	t.Run("filter by subject_contains", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?subject_contains=Hell", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 1 {
			t.Errorf("expected 1, got %d", len(data))
		}
	})

	t.Run("filter by subject exact", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?subject=World", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 1 {
			t.Errorf("expected 1, got %d", len(data))
		}
	})

	t.Run("filter by subject_regex", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?subject_regex=^(Hello|World)$", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 2 {
			t.Errorf("expected 2, got %d", len(data))
		}
	})

	t.Run("filter by body_contains", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?body_contains=123456", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 1 {
			t.Errorf("expected 1, got %d", len(data))
		}
	})

	t.Run("filter by since", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?since=1000.5", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 2 { // f-uuid2 (1001) and f-uuid3 (1002) are >= 1000.5
			t.Errorf("expected 2, got %d", len(data))
		}
	})

	t.Run("filter by until", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?until=1000.5", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 1 { // only f-uuid1 (1000.0) is <= 1000.5
			t.Errorf("expected 1, got %d", len(data))
		}
	})

	t.Run("order asc", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?order=asc", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) < 2 {
			t.Fatalf("expected at least 2 items")
		}
		// First item should have the smallest timestamp.
		first := data[0].(map[string]any)
		last := data[len(data)-1].(map[string]any)
		if first["timestamp"].(float64) > last["timestamp"].(float64) {
			t.Error("expected ascending order")
		}
	})

	t.Run("pagination limit offset", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages?limit=1&offset=1", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) != 1 {
			t.Errorf("expected 1, got %d", len(data))
		}
		meta := resp["meta"].(map[string]any)
		if meta["total"].(float64) != 3 {
			t.Errorf("meta.total = %v, want 3", meta["total"])
		}
	})
}

// TestSMTPAPI_Stats verifies the stats endpoint.
func TestSMTPAPI_Stats(t *testing.T) {
	mux, store, _ := setupAPITest(t)

	t.Run("empty", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/stats", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["count"].(float64) != 0 {
			t.Errorf("count = %v, want 0", resp["count"])
		}
		if resp["last_received_at"] != nil {
			t.Errorf("last_received_at should be nil for empty store, got %v", resp["last_received_at"])
		}
	})

	storeSMTPEvent(t, store, "st-uuid1", ParsedEmail{Subject: "Test"}, 1000.0, "default")

	t.Run("one message", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/stats", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["count"].(float64) != 1 {
			t.Errorf("count = %v, want 1", resp["count"])
		}
		if resp["last_received_at"] == nil {
			t.Error("last_received_at should not be nil")
		}
	})
}

// TestSMTPAPI_Raw verifies the raw RFC822 endpoint.
func TestSMTPAPI_Raw(t *testing.T) {
	mux, store, _ := setupAPITest(t)

	rawSrc := "From: sender@example.com\r\nTo: user@example.com\r\nSubject: Test\r\n\r\nBody"
	storeSMTPEvent(t, store, "raw-uuid1", ParsedEmail{
		Subject: "Test",
		Raw:     rawSrc,
	}, 1000.0, "default")

	t.Run("found", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/message/raw-uuid1/raw", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}
		if ct := w.Header().Get("Content-Type"); ct != "message/rfc822" {
			t.Errorf("Content-Type = %q, want %q", ct, "message/rfc822")
		}
		if w.Body.String() != rawSrc {
			t.Errorf("body = %q, want %q", w.Body.String(), rawSrc)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/message/nonexistent/raw", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", w.Code)
		}
	})
}

// TestSMTPAPI_Links verifies link extraction from HTML and plain text.
func TestSMTPAPI_Links(t *testing.T) {
	mux, store, _ := setupAPITest(t)

	storeSMTPEvent(t, store, "lnk-uuid1", ParsedEmail{
		Subject: "Test",
		HTML:    `<a href="https://example.com/reset">Reset Password</a>`,
		Text:    "Visit https://example.com or https://other.com",
	}, 1000.0, "default")

	r := httptest.NewRequest("GET", "/api/smtp/message/lnk-uuid1/links", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	// HTML: 1 link; Text: 2 unique links (example.com already seen, other.com new)
	if len(data) != 3 {
		t.Errorf("expected 3 links, got %d: %v", len(data), data)
	}

	// Verify the HTML link has anchor text.
	htmlLink := data[0].(map[string]any)
	if htmlLink["source"] != "html" {
		t.Errorf("first link source = %v, want html", htmlLink["source"])
	}
	if htmlLink["text"] != "Reset Password" {
		t.Errorf("link text = %v, want Reset Password", htmlLink["text"])
	}
}

// TestSMTPAPI_Links_NoDuplicates verifies that the same URL in HTML and text
// is deduplicated (HTML wins).
func TestSMTPAPI_Links_NoDuplicates(t *testing.T) {
	mux, store, _ := setupAPITest(t)

	storeSMTPEvent(t, store, "dup-uuid1", ParsedEmail{
		Subject: "Test",
		HTML:    `<a href="https://example.com">Click</a>`,
		Text:    "https://example.com",
	}, 1000.0, "default")

	r := httptest.NewRequest("GET", "/api/smtp/message/dup-uuid1/links", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	var resp map[string]any
	json.NewDecoder(w.Body).Decode(&resp)
	data := resp["data"].([]any)
	if len(data) != 1 {
		t.Errorf("expected 1 unique link, got %d", len(data))
	}
}

// TestSMTPAPI_Codes verifies OTP code extraction.
func TestSMTPAPI_Codes(t *testing.T) {
	mux, store, _ := setupAPITest(t)

	storeSMTPEvent(t, store, "code-uuid1", ParsedEmail{
		Subject: "Your OTP",
		Text:    "Your code is 123456. Do not share it.",
	}, 1000.0, "default")

	t.Run("default pattern", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/message/code-uuid1/codes", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}
		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		found := false
		for _, c := range data {
			if c.(string) == "123456" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected to find 123456 in %v", data)
		}
	})

	t.Run("custom pattern", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/message/code-uuid1/codes?pattern=\\d{6}", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		if len(data) == 0 {
			t.Error("expected at least one code with custom pattern")
		}
	})

	t.Run("invalid pattern", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/message/code-uuid1/codes?pattern=[invalid", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("html body codes", func(t *testing.T) {
		storeSMTPEvent(t, store, "code-uuid2", ParsedEmail{
			Subject: "HTML OTP",
			HTML:    "<p>Your verification code: <strong>654321</strong></p>",
		}, 1001.0, "default")

		r := httptest.NewRequest("GET", "/api/smtp/message/code-uuid2/codes", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		data := resp["data"].([]any)
		found := false
		for _, c := range data {
			if c.(string) == "654321" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected to find 654321 in %v", data)
		}
	})
}

// TestSMTPAPI_Delete verifies the delete endpoint.
func TestSMTPAPI_Delete(t *testing.T) {
	mux, store, _ := setupAPITest(t)
	ctx := context.Background()

	storeSMTPEvent(t, store, "del-uuid1", ParsedEmail{Subject: "Delete me"}, 1000.0, "default")
	storeSMTPEvent(t, store, "del-uuid2", ParsedEmail{Subject: "Keep me"}, 1001.0, "other")

	t.Run("delete by project", func(t *testing.T) {
		r := httptest.NewRequest("DELETE", "/api/smtp/messages?project=default", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}

		all, _ := store.FindAll(ctx, event.FindOptions{Type: "smtp"})
		if len(all) != 1 {
			t.Errorf("expected 1 remaining, got %d", len(all))
		}
		if all[0].UUID != "del-uuid2" {
			t.Errorf("wrong event remaining: %s", all[0].UUID)
		}
	})

	t.Run("delete all", func(t *testing.T) {
		r := httptest.NewRequest("DELETE", "/api/smtp/messages", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("status = %d", w.Code)
		}

		all, _ := store.FindAll(ctx, event.FindOptions{Type: "smtp"})
		if len(all) != 0 {
			t.Errorf("expected 0, got %d", len(all))
		}
	})
}

// TestSMTPAPI_Wait verifies the long-poll wait endpoint.
func TestSMTPAPI_Wait(t *testing.T) {
	mux, store, mod := setupAPITest(t)

	t.Run("timeout", func(t *testing.T) {
		r := httptest.NewRequest("GET", "/api/smtp/messages/wait?timeout=50ms&to=nobody@example.com", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != http.StatusRequestTimeout {
			t.Errorf("expected 408, got %d", w.Code)
		}
	})

	t.Run("existing matching event", func(t *testing.T) {
		storeSMTPEvent(t, store, "wait-uuid1", ParsedEmail{
			Subject: "Existing",
			To:      []EmailAddress{{Email: "wait-user@example.com"}},
		}, tsAgo(time.Second), "default")

		r := httptest.NewRequest("GET", "/api/smtp/messages/wait?to=wait-user@example.com&timeout=1s", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("new event arrives", func(t *testing.T) {
		cursor := tsNow()

		resultCh := make(chan *httptest.ResponseRecorder, 1)
		go func() {
			url := fmt.Sprintf("/api/smtp/messages/wait?to=newuser@example.com&timeout=5s&since=%.6f", cursor)
			r := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			resultCh <- w
		}()

		// Give the goroutine time to subscribe before the event arrives.
		time.Sleep(30 * time.Millisecond)

		newEv := event.Event{
			UUID: "wait-uuid2",
			Type: "smtp",
			Payload: mustMarshalJSON(ParsedEmail{
				Subject: "New Arrival",
				To:      []EmailAddress{{Email: "newuser@example.com"}},
			}),
			Timestamp: tsNow(),
			Project:   "default",
		}
		store.Store(context.Background(), newEv)
		mod.OnEventStored(newEv)

		select {
		case w := <-resultCh:
			if w.Code != 200 {
				t.Errorf("expected 200, got %d", w.Code)
			}
		case <-time.After(5 * time.Second):
			t.Error("timed out waiting for result")
		}
	})
}

// TestParseTimestamp verifies timestamp parsing from various formats.
func TestParseTimestamp(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"1700000000", 1700000000},
		{"1700000000000", 1700000000}, // unix ms
		{"1.5", 1.5},
		{"2023-11-14T22:13:20Z", 1700000000},
	}
	for _, tt := range tests {
		got := parseTimestamp(tt.input)
		if got != tt.want {
			t.Errorf("parseTimestamp(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}

	// Bad input returns 0.
	if parseTimestamp("not-a-timestamp") != 0 {
		t.Error("expected 0 for invalid input")
	}
}

// TestMatchesFilter verifies in-memory filter logic.
func TestMatchesFilter(t *testing.T) {
	makeEvent := func(email ParsedEmail, ts float64) event.Event {
		payload, _ := json.Marshal(email)
		return event.Event{
			UUID:      "test",
			Type:      "smtp",
			Payload:   payload,
			Timestamp: ts,
			Project:   "default",
		}
	}

	t.Run("to match", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{
			To: []EmailAddress{{Email: "alice@example.com", Name: "Alice"}},
		}, 1000)
		if !matchesFilter(ev, MessageFilter{To: "alice"}) {
			t.Error("expected match on partial email")
		}
		if !matchesFilter(ev, MessageFilter{To: "Alice"}) {
			t.Error("expected case-insensitive name match")
		}
		if matchesFilter(ev, MessageFilter{To: "bob"}) {
			t.Error("expected no match")
		}
	})

	t.Run("from match", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{
			From: []EmailAddress{{Email: "sender@example.com"}},
		}, 1000)
		if !matchesFilter(ev, MessageFilter{From: "sender"}) {
			t.Error("expected match")
		}
	})

	t.Run("cc match", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{
			Cc: []EmailAddress{{Email: "cc@example.com"}},
		}, 1000)
		if !matchesFilter(ev, MessageFilter{Cc: "cc@"}) {
			t.Error("expected match")
		}
	})

	t.Run("subject exact", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{Subject: "Hello World"}, 1000)
		if !matchesFilter(ev, MessageFilter{Subject: "Hello World"}) {
			t.Error("expected exact match")
		}
		if matchesFilter(ev, MessageFilter{Subject: "Hello"}) {
			t.Error("exact match should not match partial subject")
		}
	})

	t.Run("subject_contains", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{Subject: "Hello World"}, 1000)
		if !matchesFilter(ev, MessageFilter{SubjectContains: "World"}) {
			t.Error("expected contains match")
		}
	})

	t.Run("subject_regex", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{Subject: "OTP: 123456"}, 1000)
		if !matchesFilter(ev, MessageFilter{SubjectRegex: `OTP: \d+`}) {
			t.Error("expected regex match")
		}
		if matchesFilter(ev, MessageFilter{SubjectRegex: `^[invalid`}) {
			t.Error("invalid regex should not match")
		}
	})

	t.Run("body_contains text", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{Text: "Reset link here"}, 1000)
		if !matchesFilter(ev, MessageFilter{BodyContains: "Reset"}) {
			t.Error("expected text body match")
		}
	})

	t.Run("body_contains html", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{HTML: "<p>Click to verify</p>"}, 1000)
		if !matchesFilter(ev, MessageFilter{BodyContains: "verify"}) {
			t.Error("expected html body match")
		}
	})

	t.Run("since", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{}, 1000)
		if !matchesFilter(ev, MessageFilter{Since: 999}) {
			t.Error("expected match (ts >= since)")
		}
		if matchesFilter(ev, MessageFilter{Since: 1001}) {
			t.Error("expected no match (ts < since)")
		}
	})

	t.Run("until", func(t *testing.T) {
		ev := makeEvent(ParsedEmail{}, 1000)
		if !matchesFilter(ev, MessageFilter{Until: 1001}) {
			t.Error("expected match (ts <= until)")
		}
		if matchesFilter(ev, MessageFilter{Until: 999}) {
			t.Error("expected no match (ts > until)")
		}
	})
}

// TestExtractLinks verifies link extraction from HTML and text.
func TestExtractLinks(t *testing.T) {
	email := &ParsedEmail{
		HTML: `<a href="https://example.com/reset">Reset</a> <a href="https://other.com">Other</a>`,
		Text: "See https://text.com for details",
	}
	links := extractLinks(email)

	if len(links) != 3 {
		t.Fatalf("expected 3 links, got %d: %+v", len(links), links)
	}

	if links[0].Source != "html" || links[0].URL != "https://example.com/reset" {
		t.Errorf("link[0] = %+v", links[0])
	}
	if links[0].Text != "Reset" {
		t.Errorf("link[0].Text = %q, want Reset", links[0].Text)
	}
	if links[2].Source != "text" || links[2].URL != "https://text.com" {
		t.Errorf("link[2] = %+v", links[2])
	}
}

// TestSessionAuthPlain verifies project extraction from SMTP AUTH.
func TestSessionAuthPlain(t *testing.T) {
	tests := []struct {
		username string
		want     string
	}{
		{"test-run-42@smtp", "test-run-42"},
		{"myproject", "myproject"},
		{"", ""},
		{"proj@host.com", "proj"},
	}
	for _, tt := range tests {
		s := &session{}
		s.AuthPlain(tt.username, "password")
		if s.project != tt.want {
			t.Errorf("AuthPlain(%q) project = %q, want %q", tt.username, s.project, tt.want)
		}
	}
}

func mustMarshalJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

// tsNow returns the current time as a unix float (matching event.Timestamp).
func tsNow() float64 {
	return float64(time.Now().UnixMicro()) / 1_000_000
}

// tsAgo returns a timestamp N seconds before now.
func tsAgo(d time.Duration) float64 {
	return float64(time.Now().Add(-d).UnixMicro()) / 1_000_000
}
