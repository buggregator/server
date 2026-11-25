package explain

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// sdkEvent reads the "event" item of a request recorded from a real Sentry
// SDK (modules/sentry/testdata/sdk): the payload Buggregator stores.
func sdkEvent(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "sentry", "testdata", "sdk", name))
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
		BodyB64 string            `json:"body_b64"`
	}
	if err := json.Unmarshal(raw, &fx); err != nil {
		t.Fatal(err)
	}
	body := []byte(fx.Body)
	if fx.BodyB64 != "" {
		if body, err = base64.StdEncoding.DecodeString(fx.BodyB64); err != nil {
			t.Fatal(err)
		}
	}
	if fx.Headers["content-encoding"] == "gzip" {
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if body, err = io.ReadAll(zr); err != nil {
			t.Fatal(err)
		}
	}
	lines := strings.Split(string(body), "\n")
	for i := 0; i+1 < len(lines); i++ {
		var h struct {
			Type string `json:"type"`
		}
		if json.Unmarshal([]byte(lines[i]), &h) == nil && h.Type == "event" {
			return json.RawMessage(lines[i+1])
		}
	}
	t.Fatalf("%s: no event item", name)
	return nil
}

func TestPrompt_SDKEvents(t *testing.T) {
	tests := []struct {
		fixture string
		want    []string
		notWant []string
	}{
		{
			fixture: "node/02-event.json",
			want:    []string{"Error: Error: Order 15 not found", "Stack (most recent call first):", "    > ", "Last steps before the error:"},
		},
		{
			// sentry-go: exceptions as a bare array, a wrapped chain.
			fixture: "go/01-event.json",
			want:    []string{"Error: *fmt.wrapError: load order: order 15 not found", "Caused by: *main.OrderError: order 15 not found"},
		},
		{
			// Browser: the request has headers; only the method and URL go.
			fixture: "browser/02-event.json",
			want:    []string{"Error: Error: Order 15 not found", "Request: "},
			notWant: []string{"User-Agent", "Mozilla"},
		},
		{
			// sentry-laravel, gzip: a PHP job failure.
			fixture: "laravel-queue/03-event.json",
			want:    []string{"Error: RuntimeException", "Platform: php", "    > 24 |             throw new \\RuntimeException"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.fixture, func(t *testing.T) {
			text, err := Prompt(sdkEvent(t, tt.fixture), "", 0)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(text, "Explain this error") {
				t.Errorf("text does not start with the request:\n%s", text)
			}
			for _, w := range tt.want {
				if !strings.Contains(text, w) {
					t.Errorf("missing %q in:\n%s", w, text)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(text, w) {
					t.Errorf("unexpected %q in:\n%s", w, text)
				}
			}
			if len(text) > defaultTextBytes {
				t.Errorf("text is %d bytes, over %d", len(text), defaultTextBytes)
			}
		})
	}
}

func TestPrompt_QuestionAndLimit(t *testing.T) {
	payload := sdkEvent(t, "node/02-event.json")
	text, err := Prompt(payload, "Why only on Mondays?", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Question: Why only on Mondays?\n") {
		t.Errorf("question missing:\n%s", text)
	}
	for _, limit := range []int{200, 333} {
		text, _ := Prompt(payload, "Почему всё сломалось?", limit)
		if len(text) > limit || !utf8.ValidString(text) || !strings.HasSuffix(text, "[cut to fit]") {
			t.Errorf("limit %d: got %d bytes, valid=%v, %q", limit, len(text), utf8.ValidString(text), text)
		}
	}
}

func TestPrompt_NotSentry(t *testing.T) {
	if _, err := Prompt(json.RawMessage(`"just a string"`), "", 0); err == nil {
		t.Error("want an error for a payload that is not a Sentry event")
	}
}

func TestPrompt_MessageEventStack(t *testing.T) {
	stack := `{"frames":[{"filename":"main.go","function":"pay","lineno":12,"in_app":true}]}`
	tests := []struct {
		name    string
		payload json.RawMessage
	}{
		{"sentry-go threads list", sdkEvent(t, "go/02-event.json")},
		{"threads values", json.RawMessage(`{"message":"slow","threads":{"values":[{"id":1,"stacktrace":null},{"id":2,"crashed":true,"stacktrace":` + stack + `}]}}`)},
		{"top-level stacktrace", json.RawMessage(`{"message":"slow","stacktrace":` + stack + `}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := Prompt(tt.payload, "", 0)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text, "Message: ") || !strings.Contains(text, "Stack (most recent call first):\n  at ") {
				t.Errorf("no message stack in:\n%s", text)
			}
		})
	}
	text, _ := Prompt(json.RawMessage(`{"message":"no stack"}`), "", 0)
	if strings.Contains(text, "Stack") {
		t.Errorf("stack for an event without one:\n%s", text)
	}
}
