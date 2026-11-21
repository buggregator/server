package smtp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
)

// fakeRelay is an outgoing SMTP server that records what it is sent.
type fakeRelay struct {
	addr string
	user string // AUTH PLAIN credentials it accepts; "" = no AUTH offered
	pass string

	mu   sync.Mutex
	got  []relayedMail
	tlsd bool // the last message came over TLS
}

type relayedMail struct {
	from string
	to   []string
	data string
	user string
}

func startFakeRelay(t *testing.T, user, pass string, withTLS bool) *fakeRelay {
	t.Helper()
	f := &fakeRelay{user: user, pass: pass}
	s := gosmtp.NewServer(fakeBackend{f})
	s.Domain = "relay.test"
	s.AllowInsecureAuth = true
	if withTLS {
		s.TLSConfig = &tls.Config{Certificates: []tls.Certificate{selfSigned(t)}}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	f.addr = ln.Addr().String()
	go s.Serve(ln) //nolint:errcheck
	t.Cleanup(func() { s.Close() })
	return f
}

func (f *fakeRelay) mails() []relayedMail {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]relayedMail(nil), f.got...)
}

type fakeBackend struct{ f *fakeRelay }

func (b fakeBackend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	return &fakeSession{f: b.f, conn: c}, nil
}

type fakeSession struct {
	f    *fakeRelay
	conn *gosmtp.Conn
	user string
	m    relayedMail
}

func (s *fakeSession) AuthMechanisms() []string {
	if s.f.user == "" {
		return nil
	}
	return []string{sasl.Plain}
}

func (s *fakeSession) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, user, pass string) error {
		if user != s.f.user || pass != s.f.pass {
			return errors.New("bad credentials")
		}
		s.user = user
		return nil
	}), nil
}

func (s *fakeSession) Mail(from string, _ *gosmtp.MailOptions) error {
	if s.f.user != "" && s.user == "" {
		return errors.New("auth required")
	}
	s.m = relayedMail{from: from, user: s.user}
	return nil
}
func (s *fakeSession) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	s.m.to = append(s.m.to, to)
	return nil
}
func (s *fakeSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.m.data = string(b)
	s.f.mu.Lock()
	s.f.got = append(s.f.got, s.m)
	_, s.f.tlsd = s.conn.TLSConnectionState()
	s.f.mu.Unlock()
	return nil
}
func (s *fakeSession) Reset()        {}
func (s *fakeSession) Logout() error { return nil }

func selfSigned(t *testing.T) tls.Certificate {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "relay.test"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

const caughtRaw = "From: Shop <noreply@shop.test>\r\n" +
	"To: Customer <customer@example.com>,\r\n Other <other@example.com>\r\n" +
	"Cc: boss@example.com\r\n" +
	"Subject: Your order\r\n" +
	"Content-Type: text/plain\r\n" +
	"\r\n" +
	"Thanks for your order.\r\nTo: this line is body, not a header\r\n"

// relayAPI is a mux with one caught message "m1" and a relay to f.
func relayAPI(t *testing.T, cfg RelayConfig) *http.ServeMux {
	t.Helper()
	mux, store, mod := setupAPITest(t)
	mod.SetRelay(cfg)
	mux = http.NewServeMux()
	mod.RegisterProtectedRoutes(mux, func(next http.Handler) http.Handler { return next })
	storeSMTPEvent(t, store, "m1", ParsedEmail{
		Subject: "Your order",
		From:    []EmailAddress{{Email: "noreply@shop.test", Name: "Shop"}},
		To:      []EmailAddress{{Email: "customer@example.com"}},
		Raw:     caughtRaw,
	}, 1, "default")
	return mux
}

func release(mux *http.ServeMux, uuid, body string, hdr ...string) (int, map[string]any) {
	req := httptest.NewRequest("POST", "http://127.0.0.1:8000/api/smtp/release/"+uuid, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:41000" // this machine, unless a test says otherwise
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(hdr); i += 2 {
		switch hdr[i] {
		case "RemoteAddr":
			req.RemoteAddr = hdr[i+1]
		case "Host":
			req.Host = hdr[i+1]
		default:
			req.Header.Set(hdr[i], hdr[i+1])
		}
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	var m map[string]any
	json.Unmarshal(w.Body.Bytes(), &m)
	return w.Code, m
}

func relayCfg(f *fakeRelay, tlsMode string) RelayConfig {
	host, port, _ := net.SplitHostPort(f.addr)
	var p int
	for _, c := range port {
		p = p*10 + int(c-'0')
	}
	return RelayConfig{Host: host, Port: p, TLS: tlsMode, Username: f.user, Password: f.pass, InsecureSkipVerify: true,
		AllowedRecipients: []string{"*@example.com"}} // without auth release needs a list
}

func TestRelease_Sends(t *testing.T) {
	for _, tc := range []struct {
		name      string
		user      string
		withTLS   bool
		tlsMode   string
		rewrite   bool
		from      string // RelayConfig.From
		wantFrom  string
		wantTLS   bool
		wantTo    string // To header the recipient sees
		wantNoCc  bool
		recipient string
	}{
		{"as captured, plain, no auth", "", false, "none", false, "", "noreply@shop.test", false,
			"To: Customer <customer@example.com>,\r\n Other <other@example.com>", false, "me@example.com"},
		{"auth on loopback without TLS", "relay-user", false, "none", false, "", "noreply@shop.test", false,
			"To: Customer <customer@example.com>,", false, "me@example.com"},
		{"STARTTLS + auth, rewrite To, own From", "relay-user", true, "starttls", true, "qa@company.test", "qa@company.test", true,
			"To: Me@example.com, two@example.com\r\n", true, "Me <Me@EXAMPLE.com>, two@example.com"},
		{"login is an address: it is the sender", "bot@relay.test", true, "starttls", false, "", "bot@relay.test", true,
			"To: Customer <customer@example.com>,", false, "me@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := startFakeRelay(t, tc.user, "s3cret", tc.withTLS)
			cfg := relayCfg(f, tc.tlsMode)
			cfg.From = tc.from
			mux := relayAPI(t, cfg)

			body, _ := json.Marshal(map[string]any{"to": []string{tc.recipient}, "rewrite_to": tc.rewrite})
			code, resp := release(mux, "m1", string(body), "Sec-Fetch-Site", "same-origin")
			if code != 200 || resp["status"] != true {
				t.Fatalf("release = %d %v", code, resp)
			}
			got := f.mails()
			if len(got) != 1 {
				t.Fatalf("relay got %d mails", len(got))
			}
			m := got[0]
			if m.from != tc.wantFrom || m.user != tc.user || f.tlsd != tc.wantTLS {
				t.Errorf("from %q user %q tls %v", m.from, m.user, f.tlsd)
			}
			wantRcpt := strings.Split(strings.NewReplacer("Me <", "", ">", "", " ", "", "EXAMPLE", "example").Replace(tc.recipient), ",")
			if strings.Join(m.to, ",") != strings.Join(wantRcpt, ",") {
				t.Errorf("rcpt = %v, want %v", m.to, wantRcpt)
			}
			if !strings.Contains(m.data, tc.wantTo) {
				t.Errorf("To header: want %q in\n%s", tc.wantTo, m.data)
			}
			if strings.Contains(m.data, "Cc: boss@example.com") == tc.wantNoCc {
				t.Errorf("Cc kept = %v, want %v", !tc.wantNoCc, !tc.wantNoCc)
			}
			if !strings.HasSuffix(m.data, "\r\nThanks for your order.\r\nTo: this line is body, not a header\r\n") ||
				!strings.Contains(m.data, "Subject: Your order\r\n") {
				t.Errorf("body or headers changed:\n%s", m.data)
			}
		})
	}
}

func TestRelease_Guards(t *testing.T) {
	f := startFakeRelay(t, "", "", false)
	ok := `{"to":["me@example.com"]}`
	for _, tc := range []struct {
		name string
		cfg  func(c *RelayConfig)
		uuid string
		body string
		hdr  []string
		code int
	}{
		{"relay off", func(c *RelayConfig) { c.Host = "" }, "m1", ok, nil, 409},
		{"cross-site page", nil, "m1", ok, []string{"Sec-Fetch-Site", "cross-site", "Origin", "https://evil.example"}, 403},
		{"other localhost port", nil, "m1", ok, []string{"Sec-Fetch-Site", "same-site", "Origin", "http://localhost:5173"}, 403},
		{"form post", nil, "m1", ok, []string{"Content-Type", "text/plain"}, 415},
		{"bad json", nil, "m1", `{"to":`, nil, 400},
		{"no recipients", nil, "m1", `{"to":[]}`, nil, 422},
		{"not an address", nil, "m1", `{"to":["nope"]}`, nil, 422},
		{"header injection", nil, "m1", `{"to":["a@example.com\r\nBcc: x@evil.test"]}`, nil, 422},
		{"not allowed", func(c *RelayConfig) { c.AllowedRecipients = []string{"*@company.test"} }, "m1", ok, nil, 422},
		{"allowed", func(c *RelayConfig) { c.AllowedRecipients = []string{"*@company.test", "*@EXAMPLE.com"} }, "m1", ok, nil, 200},
		{"too many", nil, "m1", `{"to":["` + manyAddrs(21) + `"]}`, nil, 422},
		{"unknown message", nil, "nope", ok, nil, 404},
		{"password over plain to a remote host", func(c *RelayConfig) { c.Host = "smtp.example.com"; c.Username = "u" }, "m1", ok, nil, 502},
		{"no auth: LAN peer", nil, "m1", ok, []string{"RemoteAddr", "192.168.1.50:41000", "Host", "192.168.1.20:8000"}, 403},
		{"no auth: Docker bridge", nil, "m1", ok, []string{"RemoteAddr", "172.17.0.1:41000", "Host", "localhost:8000"}, 403},
		{"no auth: local proxy with X-Forwarded-For", nil, "m1", ok, []string{"X-Forwarded-For", "203.0.113.9"}, 403},
		{"no auth: DNS rebinding Host", nil, "m1", ok, []string{"Host", "evil.example:8000"}, 403},
		{"auth on: LAN peer", func(c *RelayConfig) { c.AuthOn = true }, "m1", ok, []string{"RemoteAddr", "192.168.1.50:41000", "Host", "192.168.1.20:8000"}, 200},
		{"no auth, no allowed_recipients", func(c *RelayConfig) { c.AllowedRecipients = nil }, "m1", ok, nil, 409},
		{"no auth, only blank patterns", func(c *RelayConfig) { c.AllowedRecipients = []string{" "} }, "m1", ok, nil, 409},
		{"auth on, no allowed_recipients", func(c *RelayConfig) { c.AuthOn = true; c.AllowedRecipients = nil }, "m1", `{"to":["anyone@anywhere.test"]}`, nil, 200},
		{"no auth, match-all pattern", func(c *RelayConfig) { c.AllowedRecipients = []string{"*"} }, "m1", ok, nil, 409},
		{"no auth, wildcard domain next to a good one", func(c *RelayConfig) { c.AllowedRecipients = []string{"*@example.com", "*@*"} }, "m1", ok, nil, 409},
		{"auth on, match-all pattern", func(c *RelayConfig) { c.AuthOn = true; c.AllowedRecipients = []string{"*"} }, "m1", ok, nil, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := relayCfg(f, "none")
			if tc.cfg != nil {
				tc.cfg(&cfg)
			}
			mux, store, mod := setupAPITest(t)
			mod.SetRelay(cfg)
			if cfg.Host == "smtp.example.com" {
				// Resolve the remote name to the fake relay.
				mod.relay.dial = func(ctx context.Context, n, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, n, f.addr)
				}
			}
			mux = http.NewServeMux()
			mod.RegisterProtectedRoutes(mux, func(next http.Handler) http.Handler { return next })
			storeSMTPEvent(t, store, "m1", ParsedEmail{Raw: caughtRaw, From: []EmailAddress{{Email: "noreply@shop.test"}}}, 1, "default")

			code, resp := release(mux, tc.uuid, tc.body, tc.hdr...)
			if code != tc.code {
				t.Errorf("code = %d, want %d: %v", code, tc.code, resp)
			}
			if cfg.Host == "smtp.example.com" && !strings.Contains(fmtMsg(resp), "unencrypted") {
				t.Errorf("message = %v, want the unencrypted refusal", resp["message"])
			}
		})
	}
	if n := len(f.mails()); n != 4 { // "allowed", "auth on: LAN peer", "auth on, no allowed_recipients", "auth on, match-all pattern"
		t.Errorf("relay got %d mails, want 4", n)
	}
}

func fmtMsg(resp map[string]any) string {
	m, _ := resp["message"].(string)
	return m
}

// Without auth (the default, listening on every interface) a request from
// another machine must never send: Buggregator would be an open relay
// through the owner's mail account. Found in review of #476.
func TestRelease_FromLANWithoutAuth(t *testing.T) {
	f := startFakeRelay(t, "", "", false)
	mux := relayAPI(t, relayCfg(f, "none"))
	req := httptest.NewRequest("POST", "http://192.168.1.20:8000/api/smtp/release/m1", strings.NewReader(`{"to":["victim@example.com"]}`)) // inside allowed_recipients: only the local guard stops it
	req.RemoteAddr = "192.168.1.50:41000"
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code == 200 || len(f.mails()) > 0 {
		t.Fatalf("LAN request without auth released the message: status %d, relayed %d", w.Code, len(f.mails()))
	}
}

// A login with insecure_skip_verify goes only to a relay on this machine.
func TestRelease_InsecureSkipVerifyRemote(t *testing.T) {
	f := startFakeRelay(t, "u", "p", true)
	_, store, mod := setupAPITest(t)
	cfg := relayCfg(f, "starttls")
	cfg.Host = "smtp.example.com"
	mod.SetRelay(cfg)
	mod.relay.dial = func(ctx context.Context, n, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, n, f.addr)
	}
	mux := http.NewServeMux()
	mod.RegisterProtectedRoutes(mux, func(next http.Handler) http.Handler { return next })
	storeSMTPEvent(t, store, "m1", ParsedEmail{Raw: caughtRaw}, 1, "default")
	code, resp := release(mux, "m1", `{"to":["me@example.com"]}`)
	if code != 502 || !strings.Contains(fmtMsg(resp), "insecure_skip_verify") || len(f.mails()) != 0 {
		t.Errorf("code %d %v, relayed %d", code, resp, len(f.mails()))
	}
}

func manyAddrs(n int) string {
	var s []string
	for i := 0; i < n; i++ {
		s = append(s, "u"+string(rune('a'+i))+"@example.com")
	}
	return strings.Join(s, ",")
}

func TestRelease_RateLimit(t *testing.T) {
	f := startFakeRelay(t, "", "", false)
	mux, store, mod := setupAPITest(t)
	mod.SetRelay(relayCfg(f, "none"))
	now := time.Unix(1000, 0)
	mod.relay.now = func() time.Time { return now }
	mux = http.NewServeMux()
	mod.RegisterProtectedRoutes(mux, func(next http.Handler) http.Handler { return next })
	storeSMTPEvent(t, store, "m1", ParsedEmail{Raw: caughtRaw}, 1, "default")

	for i := 0; i < relayBurst; i++ {
		if code, resp := release(mux, "m1", `{"to":["me@example.com"]}`); code != 200 {
			t.Fatalf("release %d = %d %v", i, code, resp)
		}
	}
	if code, _ := release(mux, "m1", `{"to":["me@example.com"]}`); code != 429 {
		t.Errorf("release over the limit = %d, want 429", code)
	}
	now = now.Add(relayWindow)
	if code, _ := release(mux, "m1", `{"to":["me@example.com"]}`); code != 200 {
		t.Errorf("release after a minute = %d, want 200", code)
	}
	if n := len(f.mails()); n != relayBurst+1 {
		t.Errorf("relay got %d mails, want %d", n, relayBurst+1)
	}
}

// The UI reads whether release is on; the password never comes back.
func TestRelayInfo(t *testing.T) {
	for _, tc := range []struct {
		cfg  RelayConfig
		want string
	}{
		{RelayConfig{}, `{"configured":false,"enabled":false}`},
		{RelayConfig{Host: "smtp.example.com", Username: "u", Password: "s3cret-pass"},
			`{"configured":true,"enabled":false,"reason":"without auth, release needs smtp_relay.allowed_recipients: your own addresses, like *@mycompany.com"}`},
		{RelayConfig{Host: "smtp.example.com", AllowedRecipients: []string{"*"}},
			`{"configured":true,"enabled":false,"reason":"without auth, smtp_relay.allowed_recipients cannot hold \"*\": it matches any domain. Use your own addresses, like *@mycompany.com, or turn auth on"}`},
		{RelayConfig{Host: "smtp.example.com", Username: "u", Password: "s3cret-pass", AllowedRecipients: []string{"me@gmail.com"}},
			`{"allowed_recipients":["me@gmail.com"],"auth":true,"configured":true,"enabled":true,"from":"","host":"smtp.example.com","local_only":true,"max_recipients":20,"port":587,"tls":"starttls"}`},
		{RelayConfig{Host: "smtp.example.com", Username: "u", Password: "s3cret-pass", AuthOn: true},
			`{"allowed_recipients":[],"auth":true,"configured":true,"enabled":true,"from":"","host":"smtp.example.com","local_only":false,"max_recipients":20,"port":587,"tls":"starttls"}`},
		{RelayConfig{Host: "mailpit", TLS: "none", AllowedRecipients: []string{"*@company.test"}, AuthOn: true},
			`{"allowed_recipients":["*@company.test"],"auth":false,"configured":true,"enabled":true,"from":"","host":"mailpit","local_only":false,"max_recipients":20,"port":25,"tls":"none"}`},
	} {
		_, _, mod := setupAPITest(t)
		mod.SetRelay(tc.cfg)
		mux := http.NewServeMux()
		mod.RegisterProtectedRoutes(mux, func(next http.Handler) http.Handler { return next })
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/smtp/relay", nil))
		if got := strings.TrimSpace(w.Body.String()); got != tc.want {
			t.Errorf("got  %s\nwant %s", got, tc.want)
		}
	}
}

// Without auth a pattern with a wildcard domain ("*", "*@*") would make
// Buggregator an open relay again: release stays off and says why.
// Found in review of #477.
func TestNoAuthBlock(t *testing.T) {
	for _, tc := range []struct {
		allowed []string
		auth    bool
		off     bool
	}{
		{nil, false, true},
		{[]string{" "}, false, true},
		{[]string{"*"}, false, true},
		{[]string{"*@*"}, false, true},
		{[]string{"*@*.com"}, false, true},
		{[]string{"me@gmail.?om"}, false, true},
		{[]string{"*@[a-z]*"}, false, true},
		{[]string{"*@"}, false, true},
		{[]string{"*@mycompany.com", "*"}, false, true},
		{[]string{"*@mycompany.com"}, false, false},
		{[]string{"me@gmail.com", " "}, false, false},
		{[]string{"qa+*@example.com"}, false, false},
		{[]string{"*"}, true, false},
		{nil, true, false},
	} {
		why := RelayConfig{Host: "smtp.example.com", AllowedRecipients: tc.allowed, AuthOn: tc.auth}.noAuthBlock()
		if (why != "") != tc.off {
			t.Errorf("%q auth=%v: block %q, want off=%v", tc.allowed, tc.auth, why, tc.off)
		}
	}
	why := RelayConfig{AllowedRecipients: []string{"*@*"}}.noAuthBlock()
	if !strings.Contains(why, `"*@*"`) || !strings.Contains(why, "auth") {
		t.Errorf("reason %q should name the pattern and the way out", why)
	}
}

func TestRewriteTo(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"CRLF, folded To, Cc", caughtRaw,
			"To: a@x.test\r\nFrom: Shop <noreply@shop.test>\r\nSubject: Your order\r\nContent-Type: text/plain\r\n\r\nThanks for your order.\r\nTo: this line is body, not a header\r\n"},
		{"LF only", "Subject: s\nTO: old@x.test\n\nbody\n", "To: a@x.test\nSubject: s\n\nbody\n"},
		{"no To at all", "Subject: s\r\n\r\nb", "To: a@x.test\r\nSubject: s\r\n\r\nb"},
	} {
		if got := string(rewriteTo([]byte(tc.in), []string{"a@x.test"})); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.name, got, tc.want)
		}
	}
}
