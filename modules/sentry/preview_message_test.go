package sentry

import (
	"encoding/json"
	"strings"
	"testing"
)

// A message-only error from captureMessage (Python, PHP SDKs) comes as
// logentry: its text must reach the live preview and the search text.
func TestPreview_LogentryMessage(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		want          any
	}{
		{"formatted", `{"level":"error","logentry":{"message":"Order %s failed","formatted":"Order 5 failed","params":["5"]}}`, "Order 5 failed"},
		{"template only", `{"level":"fatal","logentry":{"message":"Disk full"}}`, "Disk full"},
		{"message wins", `{"level":"error","message":"Plain","logentry":{"formatted":"Other"}}`, "Plain"},
		{"empty message falls back", `{"level":"error","message":"","logentry":{"formatted":"From logentry"}}`, "From logentry"},
		{"neither", `{"level":"error"}`, nil},
		{"object message", `{"level":"error","message":{"message":"Order %s failed","params":["7"],"formatted":"Order 7 failed"}}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &previewMapper{}
			b, err := m.ToPreview(json.RawMessage(tc.payload))
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatal(err)
			}
			if tc.name == "object message" {
				// The preview keeps the object (the UI reads it); search needs its text.
				if txt := m.ToSearchableText(json.RawMessage(tc.payload)); !strings.Contains(txt, "Order 7 failed") {
					t.Errorf("searchable text %q lacks the object message", txt)
				}
				return
			}
			if got["message"] != tc.want {
				t.Errorf("preview message = %#v, want %#v", got["message"], tc.want)
			}
			if s, ok := tc.want.(string); ok && !strings.Contains(m.ToSearchableText(json.RawMessage(tc.payload)), s) {
				t.Errorf("searchable text lacks %q", s)
			}
		})
	}
}
