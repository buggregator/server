package broadcast

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

// fixture is a request pusher-php-server 7.3.0 sent (testdata, captured
// with app key app-key, secret app-secret, app id app-1).
type fixture struct {
	Name, Method, Path, Library, Body string
}

func fixtures(t *testing.T) map[string]fixture {
	t.Helper()
	b, err := os.ReadFile("testdata/pusher-php-7.3.0.json")
	if err != nil {
		t.Fatal(err)
	}
	var list []fixture
	if err := json.Unmarshal(b, &list); err != nil {
		t.Fatal(err)
	}
	m := map[string]fixture{}
	for _, f := range list {
		m[f.Name] = f
	}
	return m
}

type fakeStore struct {
	mu  sync.Mutex
	got []event.Incoming
}

func (s *fakeStore) HandleIncoming(_ context.Context, inc *event.Incoming) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.got = append(s.got, *inc)
	return nil
}

func (s *fakeStore) events(t *testing.T) []Event {
	t.Helper()
	var out []Event
	for _, inc := range s.got {
		if inc.Type != "broadcast" {
			t.Errorf("type = %q", inc.Type)
		}
		var e Event
		if err := json.Unmarshal(inc.Payload, &e); err != nil {
			t.Fatal(err)
		}
		out = append(out, e)
	}
	return out
}

func request(f fixture) *http.Request {
	r := httptest.NewRequest(f.Method, f.Path, strings.NewReader(f.Body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Pusher-Library", f.Library)
	return r
}

// run sends one request through the module and returns the stored events
// and the answer to the client.
func run(t *testing.T, cfg Config, r *http.Request) ([]Event, *httptest.ResponseRecorder) {
	t.Helper()
	m := New(cfg)
	st := &fakeStore{}
	m.SetEventService(st)
	h := m.HTTPHandler()
	if !h.Match(r) {
		t.Fatalf("%s %s not matched", r.Method, r.URL)
	}
	in, err := h.Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	if !in.NoEvent || in.Respond == nil {
		t.Fatalf("want NoEvent with Respond, got %+v", in)
	}
	w := httptest.NewRecorder()
	in.Respond(w)
	return st.events(t), w
}

var secrets = Config{Secrets: map[string]string{"app-key": "app-secret"}}

func TestHandle_PusherPHP(t *testing.T) {
	fx := fixtures(t)
	for _, tc := range []struct {
		name  string
		check func(t *testing.T, ev []Event)
	}{
		{"trigger", func(t *testing.T, ev []Event) {
			e := ev[0]
			if len(ev) != 1 || e.AppID != "app-1" || e.Name != "OrderShipped" || strings.Join(e.Channels, ",") != "orders" ||
				e.Kinds[0] != "public" || e.Signature != "valid" || e.AuthKey != "app-key" || e.Library != "pusher-http-php 7.3.0" {
				t.Errorf("%+v", ev)
			}
			if string(e.DataJSON) != `{"id":42,"total":9.5}` || e.Size != len(e.Data) || len(e.Warnings) != 0 || e.Upstream != nil {
				t.Errorf("data %s size %d warnings %v", e.DataJSON, e.Size, e.Warnings)
			}
		}},
		{"trigger_multi", func(t *testing.T, ev []Event) {
			e := ev[0]
			if len(ev) != 1 || strings.Join(e.Kinds, ",") != "private,presence" || e.SocketID != "123.456" || e.Info != "subscription_count" || e.Name != `App\Events\Ping` {
				t.Errorf("%+v", e)
			}
		}},
		{"batch", func(t *testing.T, ev []Event) {
			if len(ev) != 2 || ev[0].BatchID == "" || ev[0].BatchID != ev[1].BatchID || ev[0].BatchSize != 2 {
				t.Fatalf("%+v", ev)
			}
			if ev[1].Name != "B" || ev[1].Kinds[0] != "private" || string(ev[1].DataJSON) != `"plain text"` || ev[1].Signature != "valid" {
				t.Errorf("%+v", ev[1])
			}
		}},
		{"send_to_user", func(t *testing.T, ev []Event) {
			if ev[0].Kinds[0] != "user" || ev[0].Channels[0] != "#server-to-user-7" {
				t.Errorf("%+v", ev[0])
			}
		}},
		{"encrypted", func(t *testing.T, ev []Event) {
			if !ev[0].Encrypted || ev[0].Kinds[0] != "encrypted" {
				t.Errorf("%+v", ev[0])
			}
		}},
		{"terminate", func(t *testing.T, ev []Event) {
			if len(ev) != 0 {
				t.Errorf("terminate_connections stored %d events", len(ev))
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev, w := run(t, secrets, request(fx[tc.name]))
			if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "{}" || w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("answer %d %q", w.Code, w.Body.String())
			}
			tc.check(t, ev)
		})
	}
}

func TestSignature(t *testing.T) {
	f := fixtures(t)["trigger"]
	for _, tc := range []struct {
		name string
		cfg  Config
		mod  func(r *http.Request)
		want string
	}{
		{"no secret configured", Config{}, nil, "not_checked"},
		{"other key's secret only", Config{Secrets: map[string]string{"other": "x"}}, nil, "not_checked"},
		{"right secret", secrets, nil, "valid"},
		{"wrong secret", Config{Secrets: map[string]string{"app-key": "nope"}}, nil, "invalid"},
		{"body changed", secrets, func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(strings.Replace(f.Body, "42", "43", 1)))
		}, "invalid"},
		{"path prefix (client path option)", secrets, func(r *http.Request) { r.URL.Path = "/reverb" + r.URL.Path }, "valid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := request(f)
			if tc.mod != nil {
				tc.mod(r)
			}
			ev, _ := run(t, tc.cfg, r)
			if ev[0].Signature != tc.want {
				t.Errorf("signature = %q, want %q", ev[0].Signature, tc.want)
			}
			if hasWarning(ev[0], "signature") != (tc.want == "invalid") {
				t.Errorf("warnings %v", ev[0].Warnings)
			}
		})
	}
}

func hasWarning(e Event, word string) bool {
	for _, w := range e.Warnings {
		if strings.Contains(w, word) {
			return true
		}
	}
	return false
}

func TestMatch(t *testing.T) {
	f := fixtures(t)["trigger"]
	h := New(Config{}).HTTPHandler()
	for _, tc := range []struct {
		name   string
		method string
		path   string
		typ    string
		want   bool
	}{
		{"signed events", "POST", f.Path, "", true},
		{"signed batch", "POST", fixtures(t)["batch"].Path, "", true},
		{"with prefix", "POST", "/reverb" + f.Path, "", true},
		{"unsigned", "POST", "/apps/app-1/events", "", false},
		{"GET", "GET", f.Path, "", false},
		{"typed http-dump", "POST", f.Path, "http-dump", false},
		{"typed broadcast", "POST", f.Path, "broadcast", true},
		{"no app id", "POST", "/apps//events?auth_key=k&auth_signature=s", "", false},
		{"not apps", "POST", "/api/apps?auth_key=k&auth_signature=s", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(f.Body))
			if tc.typ != "" {
				r.Header.Set("X-Buggregator-Detected-Type", tc.typ)
			}
			if got := h.Match(r); got != tc.want {
				t.Errorf("Match = %v, want %v", got, tc.want)
			}
		})
	}
}

// With an upstream the same request goes there unchanged and the app gets
// the upstream's answer; the event records the result.
func TestForward(t *testing.T) {
	f := fixtures(t)["batch"]
	var gotPath, gotBody, gotLib string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPath, gotBody, gotLib = r.URL.RequestURI(), string(b), r.Header.Get("X-Pusher-Library")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"channels":{"orders":{"subscription_count":3}}}`))
	}))
	defer up.Close()

	ev, w := run(t, Config{Upstream: up.URL + "/"}, request(f))
	if gotPath != f.Path || gotBody != f.Body || gotLib != f.Library {
		t.Errorf("forwarded %s %q %q", gotPath, gotBody, gotLib)
	}
	if w.Code != 200 || !strings.Contains(w.Body.String(), "subscription_count") {
		t.Errorf("answer %d %s", w.Code, w.Body.String())
	}
	for _, e := range ev {
		if e.Upstream == nil || e.Upstream.Status != 200 || e.Upstream.Error != "" || e.Upstream.URL != up.URL {
			t.Errorf("upstream %+v", e.Upstream)
		}
	}

	// The upstream answers an error: the app gets it as is.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"Invalid signature"}`, http.StatusUnauthorized)
	}))
	defer bad.Close()
	ev, w = run(t, Config{Upstream: bad.URL}, request(f))
	if w.Code != 401 || ev[0].Upstream.Status != 401 {
		t.Errorf("answer %d, upstream %+v", w.Code, ev[0].Upstream)
	}
}

// A down upstream: the app gets 502, as without Buggregator; the events are
// still stored with the error.
func TestForward_UpstreamDown(t *testing.T) {
	up := httptest.NewServer(http.NotFoundHandler())
	url := up.URL
	up.Close()
	ev, w := run(t, Config{Upstream: url}, request(fixtures(t)["trigger"]))
	if w.Code != http.StatusBadGateway || !strings.Contains(w.Body.String(), "upstream") {
		t.Errorf("answer %d %s", w.Code, w.Body.String())
	}
	if len(ev) != 1 || ev[0].Upstream == nil || ev[0].Upstream.Error == "" {
		t.Errorf("%+v", ev)
	}
}

func TestWarnings(t *testing.T) {
	big := strings.Repeat("x", maxDataBytes)
	chans := make([]string, 101)
	for i := range chans {
		chans[i] = "c"
	}
	cj, _ := json.Marshal(chans)
	for _, tc := range []struct {
		name, body, word string
	}{
		{"over 10 KB", `{"name":"E","channel":"c","data":"\"` + big + `\""}`, "10 KB"},
		{"over 100 channels", `{"name":"E","channels":` + string(cj) + `,"data":"{}"}`, "100"},
		{"not JSON", `{"name":"E","channel":"c","data":"{oops"}`, "not JSON"},
		{"no channel", `{"name":"E","data":"{}"}`, "no channel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/apps/1/events?auth_key=k&auth_signature=s", strings.NewReader(tc.body))
			ev, _ := run(t, Config{}, r)
			if !hasWarning(ev[0], tc.word) {
				t.Errorf("warnings %v, want one with %q", ev[0].Warnings, tc.word)
			}
		})
	}
}

func TestPreview(t *testing.T) {
	r := httptest.NewRequest("POST", "/apps/1/events?auth_key=k&auth_signature=s",
		strings.NewReader(`{"name":"Big","channel":"orders","data":"\"`+strings.Repeat("y", 5000)+`\""}`))
	ev, _ := run(t, Config{}, r)
	raw, _ := json.Marshal(ev[0])
	pm := &previewMapper{}
	pv, err := pm.ToPreview(raw)
	if err != nil {
		t.Fatal(err)
	}
	var e Event
	_ = json.Unmarshal(pv, &e)
	if len(e.Data) != previewLimit || e.DataJSON != nil || e.Size != 5002 || e.Name != "Big" {
		t.Errorf("preview data %d, size %d", len(e.Data), e.Size)
	}
	if txt := pm.ToSearchableText(raw); !strings.HasPrefix(txt, "Big 1 orders") {
		t.Errorf("search text %q", txt[:40])
	}
}
