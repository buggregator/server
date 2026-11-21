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
	req.Header.Set("Content-Type", "application/json")
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
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
	return RelayConfig{Host: host, Port: p, TLS: tlsMode, Username: f.user, Password: f.pass, InsecureSkipVerify: true}
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
			"To: me@example.com, two@example.com\r\n", true, "Me <ME@example.com>, two@example.com"},
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
			wantRcpt := strings.Split(strings.ToLower(strings.NewReplacer("Me <", "", ">", "", " ", "").Replace(tc.recipient)), ",")
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
			if cfg.Host == "smtp.example.com" && !strings.Contains(resp["message"].(string), "unencrypted") {
				t.Errorf("message = %v, want the unencrypted refusal", resp["message"])
			}
		})
	}
	if n := len(f.mails()); n != 1 { // only "allowed" went out
		t.Errorf("relay got %d mails, want 1", n)
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
		{RelayConfig{}, `{"enabled":false}`},
		{RelayConfig{Host: "smtp.example.com", Username: "u", Password: "s3cret-pass"},
			`{"allowed_recipients":[],"auth":true,"enabled":true,"from":"","host":"smtp.example.com","max_recipients":20,"port":587,"tls":"starttls"}`},
		{RelayConfig{Host: "mailpit", TLS: "none", AllowedRecipients: []string{"*@company.test"}},
			`{"allowed_recipients":["*@company.test"],"auth":false,"enabled":true,"from":"","host":"mailpit","max_recipients":20,"port":25,"tls":"none"}`},
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
