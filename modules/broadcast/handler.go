package broadcast

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/event"
)

const (
	// maxBody bounds one request: Pusher takes 10 KB an event, 10 a batch.
	maxBody = 1 << 20
	// Pusher's own limits, reported as warnings.
	maxDataBytes = 10 * 1024
	maxChannels  = 100
)

// Event is one stored broadcast event.
type Event struct {
	AppID     string          `json:"app_id"`
	Name      string          `json:"name"`
	Channels  []string        `json:"channels"`
	Kinds     []string        `json:"channel_kinds"` // per channel: public, private, presence, encrypted, user
	Data      string          `json:"data"`          // as sent: a string, JSON in most apps
	DataJSON  json.RawMessage `json:"data_json,omitempty"`
	Encrypted bool            `json:"encrypted,omitempty"` // private-encrypted-*: data is ciphertext
	SocketID  string          `json:"socket_id,omitempty"` // sent to everyone but this socket (toOthers)
	Info      string          `json:"info,omitempty"`
	AuthKey   string          `json:"auth_key,omitempty"`
	Signature string          `json:"signature"` // valid, invalid or not_checked
	Size      int             `json:"size"`      // bytes of data
	Warnings  []string        `json:"warnings"`
	Library   string          `json:"library,omitempty"` // X-Pusher-Library
	BatchID   string          `json:"batch_id,omitempty"`
	BatchSize int             `json:"batch_size,omitempty"`
	Upstream  *Upstream       `json:"upstream,omitempty"`
}

// Upstream is the result of forwarding the request to the real server.
type Upstream struct {
	URL    string `json:"url"`
	Status int    `json:"status,omitempty"`
	TimeMs int64  `json:"time_ms"`
	Error  string `json:"error,omitempty"`
}

type handler struct {
	cfg    Config
	client *http.Client
	es     EventStorer
}

func (h *handler) Priority() int { return 37 }

// appPath splits a Pusher API path: the app id and what follows it
// ("events", "batch_events", "users/7/terminate_connections"). A prefix
// before /apps/ (the client's path option) is allowed.
func appPath(p string) (appID, rest string, ok bool) {
	i := strings.LastIndex(p, "/apps/")
	if i < 0 {
		return "", "", false
	}
	appID, rest, ok = strings.Cut(p[i+len("/apps/"):], "/")
	return appID, strings.Trim(rest, "/"), ok && appID != "" && rest != ""
}

// Match: a signed POST to a Pusher API path. Pusher clients always sign
// (auth_key, auth_signature), which keeps plain http-dumps to /apps/... out.
func (h *handler) Match(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch r.Header.Get("X-Buggregator-Detected-Type") {
	case "", "broadcast":
	default:
		return false
	}
	if _, _, ok := appPath(r.URL.Path); !ok {
		return false
	}
	q := r.URL.Query()
	return q.Get("auth_key") != "" && q.Get("auth_signature") != ""
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	// broadcast.upstream points back at Buggregator: each hop would forward
	// again until the app gives up. Stop at the second hop; the first one
	// stores the event and shows this answer as the upstream's.
	if r.Header.Get(forwardedHeader) != "" {
		return respond(http.StatusLoopDetected, []byte(`{"error":"broadcast.upstream points at Buggregator itself: set it to the real Reverb/Pusher server"}`)), nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxBody {
		return nil, errors.New("broadcast: request over 1 MB")
	}
	appID, rest, _ := appPath(r.URL.Path)

	var events []Event
	switch rest {
	case "events":
		var b wireEvent
		if err := json.Unmarshal(body, &b); err != nil {
			return nil, fmt.Errorf("broadcast: events body: %w", err)
		}
		events = []Event{b.event()}
	case "batch_events":
		var b struct {
			Batch []wireEvent `json:"batch"`
		}
		if err := json.Unmarshal(body, &b); err != nil {
			return nil, fmt.Errorf("broadcast: batch_events body: %w", err)
		}
		id := event.GenerateUUID()
		for _, w := range b.Batch {
			e := w.event()
			e.BatchID, e.BatchSize = id, len(b.Batch)
			events = append(events, e)
		}
	}

	q := r.URL.Query()
	sig := h.signature(r, body)
	for i := range events {
		e := &events[i]
		e.AppID, e.AuthKey, e.Signature = appID, q.Get("auth_key"), sig
		e.Library = r.Header.Get("X-Pusher-Library")
		e.annotate()
	}

	status, answer, up := http.StatusOK, []byte("{}"), (*Upstream)(nil)
	if h.cfg.Upstream != "" {
		up = &Upstream{URL: safeURL(h.cfg.Upstream)}
		status, answer = h.forward(r, body, up)
	}

	project := r.Header.Get("X-Buggregator-Detected-Project")
	for _, e := range events {
		e.Upstream = up
		raw, err := json.Marshal(e)
		if err != nil {
			return nil, err
		}
		if h.es == nil {
			return nil, errors.New("broadcast: no event service")
		}
		if err := h.es.HandleIncoming(r.Context(), &event.Incoming{Type: "broadcast", Payload: raw, Project: project}); err != nil {
			if up == nil {
				return nil, err
			}
			// The upstream already took it: answer the app as it did.
			slog.Error("broadcast: storing a forwarded event failed", "err", err)
		}
	}

	// The client wants 200 and JSON (Laravel throws BroadcastException
	// otherwise): {} when we are the server, the upstream's own answer when
	// forwarding, so a down Reverb fails the app as it would without us.
	return respond(status, answer), nil
}

// forwardedHeader marks a request Buggregator forwarded, to catch a loop.
const forwardedHeader = "X-Buggregator-Forwarded"

// respond answers the client without storing anything more.
func respond(status int, answer []byte) *event.Incoming {
	return &event.Incoming{Type: "broadcast", NoEvent: true, Respond: func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(answer)
	}}
}

// safeURL is the upstream as shown on a card: without user:password.
func safeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User == nil {
		return raw
	}
	u.User = nil
	return u.String()
}

// wireEvent is an event as the client sends it: one channel or several.
type wireEvent struct {
	Name     string   `json:"name"`
	Data     any      `json:"data"`
	Channel  string   `json:"channel"`
	Channels []string `json:"channels"`
	SocketID string   `json:"socket_id"`
	Info     string   `json:"info"`
}

func (w wireEvent) event() Event {
	e := Event{Name: w.Name, Channels: w.Channels, SocketID: w.SocketID, Info: w.Info}
	if w.Channel != "" {
		e.Channels = append([]string{w.Channel}, e.Channels...)
	}
	if e.Channels == nil {
		e.Channels = []string{}
	}
	switch d := w.Data.(type) {
	case string:
		e.Data = d
	case nil:
	default: // a client that sends an object, not a string
		b, _ := json.Marshal(d)
		e.Data = string(b)
	}
	return e
}

// annotate fills the derived fields and the warnings.
func (e *Event) annotate() {
	e.Size = len(e.Data)
	e.Kinds = make([]string, len(e.Channels))
	for i, c := range e.Channels {
		e.Kinds[i] = channelKind(c)
		if e.Kinds[i] == "encrypted" {
			e.Encrypted = true
		}
	}
	if json.Valid([]byte(e.Data)) {
		e.DataJSON = json.RawMessage(e.Data)
	}
	e.Warnings = []string{}
	if e.Size > maxDataBytes {
		e.Warnings = append(e.Warnings, fmt.Sprintf("data is %d bytes: over 10 KB, Pusher rejects it", e.Size))
	}
	if len(e.Channels) > maxChannels {
		e.Warnings = append(e.Warnings, fmt.Sprintf("%d channels: over 100, Pusher rejects it", len(e.Channels)))
	}
	if len(e.Channels) == 0 {
		e.Warnings = append(e.Warnings, "no channel")
	}
	if e.Data != "" && e.DataJSON == nil {
		e.Warnings = append(e.Warnings, "data is not JSON: Laravel Echo cannot parse it")
	}
	if e.Signature == "invalid" {
		e.Warnings = append(e.Warnings, "signature or body hash does not match the configured secret: the app's secret differs, or the body changed on the way")
	}
}

func channelKind(c string) string {
	switch {
	case strings.HasPrefix(c, "private-encrypted-"):
		return "encrypted"
	case strings.HasPrefix(c, "private-"):
		return "private"
	case strings.HasPrefix(c, "presence-"):
		return "presence"
	case strings.HasPrefix(c, "#server-to-user-"):
		return "user"
	}
	return "public"
}

// signature checks auth_signature: HMAC-SHA256 with the app secret over
// "POST\n<path>\n<sorted query without auth_signature>". The path signed is
// /apps/... with or without the client's prefix, so both are tried.
func (h *handler) signature(r *http.Request, body []byte) string {
	q := r.URL.Query()
	secret, ok := h.cfg.Secrets[q.Get("auth_key")]
	if !ok || secret == "" {
		return "not_checked"
	}
	if m := q.Get("body_md5"); m != "" {
		sum := md5.Sum(body)
		if hex.EncodeToString(sum[:]) != m {
			return "invalid"
		}
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		if k != "auth_signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = strings.ToLower(k) + "=" + q.Get(k)
	}
	query := strings.Join(parts, "&")
	got, _ := hex.DecodeString(q.Get("auth_signature"))
	paths := []string{r.URL.Path}
	if i := strings.LastIndex(r.URL.Path, "/apps/"); i > 0 {
		paths = append(paths, r.URL.Path[i:])
	}
	for _, p := range paths {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(r.Method + "\n" + p + "\n" + query))
		if hmac.Equal(mac.Sum(nil), got) {
			return "valid"
		}
	}
	return "invalid"
}

// forward sends the request as it came (path, query, body) to the upstream;
// the host is not signed, so the signature stays valid. It returns what to
// answer the app.
func (h *handler) forward(r *http.Request, body []byte, up *Upstream) (int, []byte) {
	start := time.Now()
	req, err := http.NewRequestWithContext(r.Context(), r.Method, h.cfg.Upstream+r.URL.RequestURI(), bytes.NewReader(body))
	if err == nil {
		for _, k := range []string{"Content-Type", "X-Pusher-Library", "User-Agent"} {
			if v := r.Header.Get(k); v != "" {
				req.Header.Set(k, v)
			}
		}
		req.Header.Set(forwardedHeader, "1")
		var resp *http.Response
		resp, err = h.client.Do(req)
		if err == nil {
			defer resp.Body.Close()
			b, rerr := io.ReadAll(io.LimitReader(resp.Body, maxBody))
			up.Status, up.TimeMs = resp.StatusCode, time.Since(start).Milliseconds()
			if rerr != nil {
				up.Error = rerr.Error()
			}
			return resp.StatusCode, b
		}
	}
	// The request URL carries the signature: keep only the cause.
	var uerr *url.Error
	if errors.As(err, &uerr) {
		err = uerr.Err
	}
	up.TimeMs, up.Error = time.Since(start).Milliseconds(), err.Error()
	slog.Warn("broadcast: forward to upstream failed", "upstream", h.cfg.Upstream, "err", err)
	msg, _ := json.Marshal(map[string]string{"error": "Buggregator could not reach the broadcast upstream: " + err.Error()})
	return http.StatusBadGateway, msg
}
