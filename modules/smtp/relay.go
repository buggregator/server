package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
)

// RelayConfig is the outgoing SMTP server a caught message can be released
// to. Off while Host is empty. Password never leaves the server.
type RelayConfig struct {
	Host               string
	Port               int    // default: 587 for starttls, 465 for tls, 25 for none
	TLS                string // "starttls" (default), "tls" (implicit, port 465) or "none"
	Username           string // empty: no AUTH
	Password           string
	From               string   // MAIL FROM; empty: Username when it is an address, else the message's From
	AllowedRecipients  []string // address patterns like "*@example.com"; empty: any
	InsecureSkipVerify bool     // accept any TLS certificate (a local test relay)

	// AuthOn is the server's API auth, not a relay setting: without it a
	// release is taken only from this machine.
	AuthOn bool
}

// Enabled reports whether a relay is configured.
func (c RelayConfig) Enabled() bool { return strings.TrimSpace(c.Host) != "" }

// needsAllowList: without auth a proxy on this host makes outside requests
// look local, so release then also needs allowed_recipients, and mail can go
// only to the owner's own addresses.
func (c RelayConfig) needsAllowList() bool {
	if c.AuthOn {
		return false
	}
	for _, p := range c.AllowedRecipients {
		if strings.TrimSpace(p) != "" {
			return false
		}
	}
	return true
}

const allowListHint = "without auth, release needs smtp_relay.allowed_recipients: your own addresses, like *@mycompany.com"

func (c RelayConfig) tlsMode() string {
	switch m := strings.ToLower(strings.TrimSpace(c.TLS)); m {
	case "tls", "none":
		return m
	default:
		return "starttls"
	}
}

func (c RelayConfig) port() int {
	if c.Port > 0 {
		return c.Port
	}
	switch c.tlsMode() {
	case "tls":
		return 465
	case "none":
		return 25
	}
	return 587
}

// relayLimit caps releases: at most relayBurst messages per relayWindow, so a
// stuck button or a script cannot flood a real mailbox.
const (
	relayBurst         = 10
	relayWindow        = time.Minute
	relayMaxRecipients = 20
	relayTimeout       = 30 * time.Second
)

var (
	errRelayOff         = errors.New("no outgoing SMTP relay is configured")
	errRelayRateLimited = errors.New("too many messages released in the last minute")
)

// relay sends caught messages through the configured outgoing server.
type relay struct {
	cfg  RelayConfig
	dial func(ctx context.Context, network, addr string) (net.Conn, error)

	mu   sync.Mutex
	sent []time.Time // release times within relayWindow
	now  func() time.Time
}

func newRelay(cfg RelayConfig) *relay {
	d := &net.Dialer{Timeout: 10 * time.Second}
	return &relay{cfg: cfg, dial: d.DialContext, now: time.Now}
}

// allow takes one slot of the rate limit.
func (r *relay) allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	keep := r.sent[:0]
	for _, t := range r.sent {
		if now.Sub(t) < relayWindow {
			keep = append(keep, t)
		}
	}
	r.sent = keep
	if len(r.sent) >= relayBurst {
		return false
	}
	r.sent = append(r.sent, now)
	return true
}

// recipients parses and checks the addresses typed by the user: valid,
// unique, at most relayMaxRecipients, each allowed by AllowedRecipients.
func (r *relay) recipients(in []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	for _, s := range in {
		for _, part := range strings.Split(s, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			a, err := mail.ParseAddress(part)
			if err != nil {
				return nil, fmt.Errorf("not an email address: %q", part)
			}
			addr := normalizeAddress(a.Address)
			if !r.recipientAllowed(addr) {
				return nil, fmt.Errorf("%s is not in the allowed recipients", addr)
			}
			if !seen[addr] {
				seen[addr] = true
				out = append(out, addr)
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no recipients")
	}
	if len(out) > relayMaxRecipients {
		return nil, fmt.Errorf("at most %d recipients", relayMaxRecipients)
	}
	return out, nil
}

// normalizeAddress lowercases the domain only: the local part is case
// sensitive by RFC 5321.
func normalizeAddress(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return addr[:i] + strings.ToLower(addr[i:])
	}
	return addr
}

// recipientAllowed matches the patterns ignoring case.
func (r *relay) recipientAllowed(addr string) bool {
	if len(r.cfg.AllowedRecipients) == 0 {
		return true
	}
	addr = strings.ToLower(addr)
	for _, p := range r.cfg.AllowedRecipients {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if ok, _ := path.Match(p, addr); ok {
			return true
		}
	}
	return false
}

// envelopeFrom is the MAIL FROM: the configured From, else the login when it
// is an address (real relays like Gmail or SES reject a sender that is not
// the account), else the captured message's From.
func (c RelayConfig) envelopeFrom(captured string) string {
	if f := strings.TrimSpace(c.From); f != "" {
		return f
	}
	if u := strings.TrimSpace(c.Username); strings.Contains(u, "@") {
		if a, err := mail.ParseAddress(u); err == nil {
			return a.Address
		}
	}
	return captured
}

// send delivers raw to the recipients. from is the captured message's From.
func (r *relay) send(ctx context.Context, from string, to []string, raw []byte) error {
	if !r.cfg.Enabled() {
		return errRelayOff
	}
	from = r.cfg.envelopeFrom(from)

	ctx, cancel := context.WithTimeout(ctx, relayTimeout)
	defer cancel()
	host := strings.TrimSpace(r.cfg.Host)
	addr := net.JoinHostPort(host, strconv.Itoa(r.cfg.port()))
	conn, err := r.dial(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}

	tlsCfg := &tls.Config{ServerName: host, InsecureSkipVerify: r.cfg.InsecureSkipVerify} //nolint:gosec // opt-in for a local test relay
	var c *gosmtp.Client
	switch r.cfg.tlsMode() {
	case "tls":
		c = gosmtp.NewClient(tls.Client(conn, tlsCfg))
	case "none":
		c = gosmtp.NewClient(conn)
	default:
		if c, err = gosmtp.NewClientStartTLS(conn, tlsCfg); err != nil {
			conn.Close()
			return fmt.Errorf("STARTTLS with %s: %w", addr, err)
		}
	}
	defer c.Close()

	if r.cfg.Username != "" {
		// The password goes in clear text, or over TLS with an unchecked
		// certificate, only to a relay on this machine.
		if _, encrypted := c.TLSConnectionState(); !encrypted && !isLoopbackHost(host) {
			return errors.New("refusing to send the password over an unencrypted connection: set tls to starttls or tls")
		}
		if r.cfg.InsecureSkipVerify && !isLoopbackHost(host) {
			return errors.New("refusing to send the password with insecure_skip_verify to a remote relay: anyone in between could take it")
		}
		if err := c.Auth(sasl.NewPlainClient("", r.cfg.Username, r.cfg.Password)); err != nil {
			return fmt.Errorf("login to %s: %w", addr, err)
		}
	}
	if err := c.SendMail(from, to, bytes.NewReader(raw)); err != nil {
		return fmt.Errorf("send via %s: %w", addr, err)
	}
	return c.Quit()
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// rewriteTo replaces the To header of raw with to and drops Cc, so the
// recipient sees the message addressed to them. The body is kept byte for
// byte.
func rewriteTo(raw []byte, to []string) []byte {
	sep := []byte("\r\n\r\n")
	nl := "\r\n"
	i := bytes.Index(raw, sep)
	if j := bytes.Index(raw, []byte("\n\n")); j >= 0 && (i < 0 || j < i) {
		sep, nl, i = []byte("\n\n"), "\n", j
	}
	head, body := raw, []byte(nil)
	if i >= 0 {
		head, body = raw[:i], raw[i:]
	}

	var kept []string
	drop := false
	for _, line := range strings.Split(string(head), nl) {
		if line != "" && (line[0] == ' ' || line[0] == '\t') {
			if !drop {
				kept = append(kept, line)
			}
			continue // a folded continuation of the previous header
		}
		name, _, _ := strings.Cut(line, ":")
		name = strings.ToLower(strings.TrimSpace(name))
		drop = name == "to" || name == "cc"
		if !drop {
			kept = append(kept, line)
		}
	}
	out := "To: " + strings.Join(to, ", ") + nl + strings.Join(kept, nl)
	return append([]byte(out), body...)
}
