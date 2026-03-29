package event

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

func TestGenerateUUID(t *testing.T) {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	t.Run("format", func(t *testing.T) {
		uuid := GenerateUUID()
		if !uuidRegex.MatchString(uuid) {
			t.Errorf("UUID %q does not match UUID v4 format", uuid)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			uuid := GenerateUUID()
			if seen[uuid] {
				t.Fatalf("duplicate UUID generated: %s", uuid)
			}
			seen[uuid] = true
		}
	})
}

func TestNewEvent(t *testing.T) {
	t.Run("generates UUID when empty", func(t *testing.T) {
		inc := &Incoming{
			Type:    "sentry",
			Payload: json.RawMessage(`{"key":"value"}`),
			Project: "default",
		}
		ev := NewEvent(inc)
		if ev.UUID == "" {
			t.Error("expected UUID to be generated")
		}
		if ev.Type != "sentry" {
			t.Errorf("Type = %q, want %q", ev.Type, "sentry")
		}
		if ev.Project != "default" {
			t.Errorf("Project = %q, want %q", ev.Project, "default")
		}
		if string(ev.Payload) != `{"key":"value"}` {
			t.Errorf("Payload = %s, want %s", ev.Payload, `{"key":"value"}`)
		}
	})

	t.Run("uses provided UUID", func(t *testing.T) {
		inc := &Incoming{
			UUID:    "my-custom-uuid",
			Type:    "ray",
			Payload: json.RawMessage(`{}`),
		}
		ev := NewEvent(inc)
		if ev.UUID != "my-custom-uuid" {
			t.Errorf("UUID = %q, want %q", ev.UUID, "my-custom-uuid")
		}
	})

	t.Run("sets timestamp", func(t *testing.T) {
		before := float64(time.Now().UnixMicro()) / 1_000_000
		ev := NewEvent(&Incoming{Type: "test", Payload: json.RawMessage(`{}`)})
		after := float64(time.Now().UnixMicro()) / 1_000_000

		if ev.Timestamp < before || ev.Timestamp > after {
			t.Errorf("Timestamp %f not in range [%f, %f]", ev.Timestamp, before, after)
		}
	})

	t.Run("IsPinned defaults to false", func(t *testing.T) {
		ev := NewEvent(&Incoming{Type: "test", Payload: json.RawMessage(`{}`)})
		if ev.IsPinned {
			t.Error("expected IsPinned to be false")
		}
	})
}

func TestPreviewRegistry(t *testing.T) {
	t.Run("Get returns nil for unregistered type", func(t *testing.T) {
		reg := NewPreviewRegistry()
		if mapper := reg.Get("unknown"); mapper != nil {
			t.Errorf("expected nil, got %v", mapper)
		}
	})

	t.Run("Register and Get", func(t *testing.T) {
		reg := NewPreviewRegistry()
		m := &mockMapper{
			previewPayload: json.RawMessage(`{"preview":true}`),
			searchText:     "search text",
		}
		reg.Register("test-type", m)

		got := reg.Get("test-type")
		if got != m {
			t.Error("expected registered mapper")
		}
	})

	t.Run("BuildPreview with mapper", func(t *testing.T) {
		reg := NewPreviewRegistry()
		reg.Register("sentry", &mockMapper{
			previewPayload: json.RawMessage(`{"preview":true}`),
			searchText:     "error in production",
		})

		ev := Event{
			UUID:      "test-uuid",
			Type:      "sentry",
			Payload:   json.RawMessage(`{"full":"payload"}`),
			Timestamp: 1234567890.123456,
			Project:   "myproject",
			IsPinned:  true,
		}

		p := reg.BuildPreview(ev)
		if p.UUID != "test-uuid" {
			t.Errorf("UUID = %q, want %q", p.UUID, "test-uuid")
		}
		if p.Type != "sentry" {
			t.Errorf("Type = %q, want %q", p.Type, "sentry")
		}
		if string(p.Payload) != `{"preview":true}` {
			t.Errorf("Payload = %s, want %s", p.Payload, `{"preview":true}`)
		}
		if p.SearchableText != "error in production" {
			t.Errorf("SearchableText = %q, want %q", p.SearchableText, "error in production")
		}
		if p.Timestamp != 1234567890.123456 {
			t.Errorf("Timestamp = %f, want %f", p.Timestamp, 1234567890.123456)
		}
		if p.Project != "myproject" {
			t.Errorf("Project = %q, want %q", p.Project, "myproject")
		}
		if !p.IsPinned {
			t.Error("expected IsPinned to be true")
		}
	})

	t.Run("BuildPreview without mapper passes through payload", func(t *testing.T) {
		reg := NewPreviewRegistry()
		ev := Event{
			UUID:    "test-uuid",
			Type:    "unknown",
			Payload: json.RawMessage(`{"original":"data"}`),
		}

		p := reg.BuildPreview(ev)
		if string(p.Payload) != `{"original":"data"}` {
			t.Errorf("Payload = %s, want original payload", p.Payload)
		}
		if p.SearchableText != "" {
			t.Errorf("SearchableText = %q, want empty", p.SearchableText)
		}
	})
}

// mockMapper implements PreviewMapper for testing.
type mockMapper struct {
	previewPayload json.RawMessage
	searchText     string
}

func (m *mockMapper) ToPreview(_ json.RawMessage) (json.RawMessage, error) {
	return m.previewPayload, nil
}

func (m *mockMapper) ToSearchableText(_ json.RawMessage) string {
	return m.searchText
}
