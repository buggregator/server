package sentry

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestParseEnvelopeItems_NewlineDelimited(t *testing.T) {
	body := []byte(`{"event_id":"abc","sent_at":"2026-01-01T00:00:00Z"}
{"type":"event"}
{"event_id":"abc","level":"error","message":"boom"}`)

	header, items, err := parseEnvelopeItems(body)
	if err != nil {
		t.Fatal(err)
	}
	if header.EventID != "abc" {
		t.Errorf("header.EventID = %q, want %q", header.EventID, "abc")
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Type != "event" {
		t.Errorf("item type = %q, want event", items[0].Type)
	}

	var p map[string]any
	if err := json.Unmarshal(items[0].Payload, &p); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if p["message"] != "boom" {
		t.Errorf("payload.message = %v, want boom", p["message"])
	}
}

// Sentry envelopes can encode payloads with explicit byte lengths so that
// payloads containing literal newlines aren't split across items.
func TestParseEnvelopeItems_HonorsLengthForMultilinePayload(t *testing.T) {
	payload := "{\n  \"message\": \"first\\nsecond\"\n}"
	body := []byte("{\"event_id\":\"e1\"}\n" +
		"{\"type\":\"event\",\"length\":" + strconv.Itoa(len(payload)) + "}\n" +
		payload)

	_, items, err := parseEnvelopeItems(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if string(items[0].Payload) != payload {
		t.Errorf("payload = %q, want %q", items[0].Payload, payload)
	}

	var p map[string]any
	if err := json.Unmarshal(items[0].Payload, &p); err != nil {
		t.Fatalf("payload not JSON: %v", err)
	}
	if p["message"] != "first\nsecond" {
		t.Errorf("payload.message = %v", p["message"])
	}
}

func TestParseEnvelopeItems_MultipleItems(t *testing.T) {
	p1 := `{"event_id":"e1","level":"error"}`
	p2 := `{"trace_id":"tt","level":"info","body":"log line"}`

	body := []byte("{\"event_id\":\"e1\"}\n" +
		"{\"type\":\"event\",\"length\":" + strconv.Itoa(len(p1)) + "}\n" +
		p1 + "\n" +
		"{\"type\":\"log\",\"length\":" + strconv.Itoa(len(p2)) + "}\n" +
		p2)

	_, items, err := parseEnvelopeItems(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].Type != "event" {
		t.Errorf("item[0].Type = %q", items[0].Type)
	}
	if items[1].Type != "log" {
		t.Errorf("item[1].Type = %q", items[1].Type)
	}
	if string(items[0].Payload) != p1 {
		t.Errorf("item[0].Payload = %q", items[0].Payload)
	}
	if string(items[1].Payload) != p2 {
		t.Errorf("item[1].Payload = %q", items[1].Payload)
	}
}

func TestParseEnvelopeItems_BadItemHeaderIsSkipped(t *testing.T) {
	// Item header is unparseable — the parser must move past the next newline
	// and not crash. Subsequent valid items should still be parsed.
	body := []byte(strings.Join([]string{
		`{"event_id":"e1"}`,
		`not-json-header`,
		`should-be-skipped`,
		`{"type":"event"}`,
		`{"message":"ok"}`,
	}, "\n"))

	_, items, err := parseEnvelopeItems(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1, got %v", len(items), items)
	}
	if items[0].Type != "event" {
		t.Errorf("item[0].Type = %q", items[0].Type)
	}
}

func TestParseEnvelopeItems_HeaderOnly(t *testing.T) {
	body := []byte(`{"event_id":"only-header"}`)
	header, items, err := parseEnvelopeItems(body)
	if err != nil {
		t.Fatal(err)
	}
	if header.EventID != "only-header" {
		t.Errorf("header.EventID = %q", header.EventID)
	}
	if len(items) != 0 {
		t.Errorf("items = %d, want 0", len(items))
	}
}

func TestParseEnvelopeItems_LengthBeyondBody_FallsBackToNewline(t *testing.T) {
	// length declares more bytes than the body has — the parser must fall back
	// to the newline-delimited path rather than crash.
	body := []byte(`{"event_id":"e1"}
{"type":"event","length":9999}
{"message":"trimmed"}`)

	_, items, err := parseEnvelopeItems(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if !strings.Contains(string(items[0].Payload), "trimmed") {
		t.Errorf("payload = %q, expected to contain 'trimmed'", items[0].Payload)
	}
}
