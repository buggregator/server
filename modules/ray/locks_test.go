package ray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// pauseBody is what spatie/ray 1.48.1 sends for ray()->pause().
const pauseBody = `{"uuid":"ev-lock","payloads":[{"type":"create_lock","content":{"name":"9b7848e9354504312ef96c53b5a188f4"},"origin":{"file":"/app/p.php","line_number":9}}],"meta":{"php_version":"8.3"}}`

const lockName = "9b7848e9354504312ef96c53b5a188f4"

// pausedModule is a ray module with a stored pause event, as after the
// client's create_lock request went through ingestion.
func pausedModule(t *testing.T) (*Module, *http.ServeMux, func() map[string]any) {
	t.Helper()
	es, store := newRayService(t)
	m := New()
	m.SetEventService(es)
	mux := http.NewServeMux()
	m.RegisterRoutes(mux, store)
	m.RegisterProtectedRoutes(mux, func(h http.Handler) http.Handler { return h })

	inc, err := m.HTTPHandler().Handle(httptest.NewRequest("POST", "/", strings.NewReader(pauseBody)))
	if err != nil {
		t.Fatal(err)
	}
	if err := es.HandleIncoming(context.Background(), inc); err != nil {
		t.Fatal(err)
	}
	content := func() map[string]any {
		ev, err := store.FindByUUID(context.Background(), "ev-lock")
		if err != nil || ev == nil {
			t.Fatalf("event: %v", err)
		}
		var doc struct {
			Payloads []struct {
				Content map[string]any `json:"content"`
			} `json:"payloads"`
		}
		if err := json.Unmarshal(ev.Payload, &doc); err != nil || len(doc.Payloads) != 1 {
			t.Fatalf("payload %s: %v", ev.Payload, err)
		}
		return doc.Payloads[0].Content
	}
	return m, mux, content
}

func do(t *testing.T, mux *http.ServeMux, method, path, body string) (int, map[string]any) {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("%s %s: not JSON: %q", method, path, w.Body.String())
	}
	return w.Code, got
}

func TestLocks_PollAndRelease(t *testing.T) {
	poll := "/locks/" + lockName + "?hostname=web&project_name="
	tests := []struct {
		name       string
		body       string
		wantState  string
		wantActive bool
		wantStop   bool
	}{
		{"continue", `{"token":"","project":"default"}`, LockContinued, false, false},
		{"stop", `{"token":"","project":"default","stop_execution":true}`, LockStopped, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mux, content := pausedModule(t)

			if code, got := do(t, mux, "GET", poll, ""); code != 200 || got["active"] != true || got["stop_execution"] != false {
				t.Fatalf("while paused: %d %v, want active", code, got)
			}

			code, got := do(t, mux, "POST", "/api/ray/locks/"+lockName, tt.body)
			if code != 200 || got["status"] != true || got["state"] != tt.wantState {
				t.Fatalf("release: %d %v, want state %s", code, got, tt.wantState)
			}

			_, got = do(t, mux, "GET", poll, "")
			if got["active"] != tt.wantActive || got["stop_execution"] != tt.wantStop {
				t.Errorf("after release: %v, want active=%v stop=%v", got, tt.wantActive, tt.wantStop)
			}
			if c := content(); c["state"] != tt.wantState || c["name"] != lockName {
				t.Errorf("event content = %v, want state %s", c, tt.wantState)
			}

			// A second click (or another tab) changes nothing.
			_, got = do(t, mux, "POST", "/api/ray/locks/"+lockName, `{"stop_execution":true}`)
			if got["state"] != tt.wantState {
				t.Errorf("second release changed state to %v", got["state"])
			}
		})
	}
}

func TestLocks_Unknown(t *testing.T) {
	_, mux, _ := pausedModule(t)
	if code, got := do(t, mux, "GET", "/locks/nope", ""); code != 200 || got["active"] != false || got["stop_execution"] != false {
		t.Errorf("unknown poll: %d %v, want inactive", code, got)
	}
	if code, _ := do(t, mux, "POST", "/api/ray/locks/nope", `{}`); code != http.StatusNotFound {
		t.Errorf("unknown release: %d, want 404", code)
	}
}

// Two pauses in a row: releasing the first leaves the second paused.
func TestLocks_TwoPauses(t *testing.T) {
	l := newLocks()
	l.create("a", "ev-1")
	l.create("b", "ev-2")
	if _, _, ok := l.release("a", false); !ok {
		t.Fatal("release a")
	}
	if st, _, _ := l.get("b"); st != LockActive {
		t.Errorf("b = %q, want active", st)
	}
}

func TestLocks_ExpireOldOnCreate(t *testing.T) {
	l := newLocks()
	now := time.Unix(1_800_000_000, 0)
	l.now = func() time.Time { return now }
	l.create("old", "ev-1")
	now = now.Add(lockTTL + time.Minute)
	l.create("new", "ev-2")
	if _, _, ok := l.get("old"); ok {
		t.Error("a lock older than lockTTL is kept")
	}
	if _, _, ok := l.get("new"); !ok {
		t.Error("new lock missing")
	}
}

func TestLockNames(t *testing.T) {
	tests := []struct {
		name, body string
		want       []string
	}{
		{"pause", pauseBody, []string{lockName}},
		{"other payload", `{"uuid":"x","payloads":[{"type":"log","content":{"values":["a"]}}]}`, nil},
		{"no name", `{"uuid":"x","payloads":[{"type":"create_lock","content":{}}]}`, nil},
		{"not json", `nope`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lockNames([]byte(tt.body))
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("lockNames = %v, want %v", got, tt.want)
			}
		})
	}
}
