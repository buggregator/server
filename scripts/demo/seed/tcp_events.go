package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// --- SMTP -----------------------------------------------------------------------

// plainAuth is AUTH PLAIN without net/smtp's TLS-or-localhost check: the
// Buggregator SMTP server accepts any credentials, and the username picks the
// project.
type plainAuth struct{ user string }

func (a plainAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + a.user + "\x00demo"), nil
}

func (a plainAuth) Next([]byte, bool) ([]byte, error) { return nil, nil }

type mail struct {
	project  string
	from     string
	to, cc   []string
	bcc      []string // envelope only, never in the headers
	subject  string
	text     string
	html     string
	attach   []attachment
	priority string
}

type attachment struct {
	name, contentType string
	data              []byte
}

func (m mail) build() []byte {
	var b bytes.Buffer
	hdr := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	hdr("From", m.from)
	hdr("To", strings.Join(m.to, ", "))
	if len(m.cc) > 0 {
		hdr("Cc", strings.Join(m.cc, ", "))
	}
	hdr("Subject", m.subject)
	hdr("Date", time.Now().Format(time.RFC1123Z))
	hdr("Message-ID", "<"+randHex(12)+"@"+m.project+".example.com>")
	hdr("MIME-Version", "1.0")
	if m.priority != "" {
		hdr("X-Priority", m.priority)
	}

	alt := "alt-" + randHex(8)
	altPart := func() {
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n%s\r\n", alt, qp(m.text))
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=utf-8\r\nContent-Transfer-Encoding: quoted-printable\r\n\r\n%s\r\n", alt, qp(m.html))
		fmt.Fprintf(&b, "--%s--\r\n", alt)
	}

	if len(m.attach) == 0 {
		hdr("Content-Type", `multipart/alternative; boundary="`+alt+`"`)
		b.WriteString("\r\n")
		altPart()
		return b.Bytes()
	}

	mixed := "mixed-" + randHex(8)
	hdr("Content-Type", `multipart/mixed; boundary="`+mixed+`"`)
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", mixed, alt)
	altPart()
	for _, a := range m.attach {
		fmt.Fprintf(&b, "--%s\r\nContent-Type: %s; name=\"%s\"\r\nContent-Disposition: attachment; filename=\"%s\"\r\nContent-Transfer-Encoding: base64\r\n\r\n",
			mixed, a.contentType, a.name, a.name)
		enc := base64.StdEncoding.EncodeToString(a.data)
		for len(enc) > 76 {
			b.WriteString(enc[:76] + "\r\n")
			enc = enc[76:]
		}
		b.WriteString(enc + "\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", mixed)
	return b.Bytes()
}

func qp(s string) string {
	var b bytes.Buffer
	w := quotedprintable.NewWriter(&b)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return b.String()
}

func addr(full string) string {
	if i := strings.Index(full, "<"); i >= 0 {
		return strings.TrimSuffix(full[i+1:], ">")
	}
	return full
}

func (s *seeder) seedSMTP() {
	if s.smtpAddr == "" {
		return
	}
	for _, m := range demoMails() {
		rcpt := []string{}
		for _, list := range [][]string{m.to, m.cc, m.bcc} {
			for _, r := range list {
				rcpt = append(rcpt, addr(r))
			}
		}
		if err := smtp.SendMail(s.smtpAddr, plainAuth{user: m.project}, addr(m.from), rcpt, m.build()); err != nil {
			s.fail("smtp "+m.subject, err)
			continue
		}
		s.count("smtp", 1)
	}
}

func emailLayout(title, body string) string {
	return `<!doctype html><html><body style="margin:0;background:#f4f5f7;font-family:Helvetica,Arial,sans-serif;color:#1f2933">` +
		`<table width="100%" cellpadding="0" cellspacing="0"><tr><td align="center" style="padding:32px 12px">` +
		`<table width="560" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;padding:32px">` +
		`<tr><td><h1 style="font-size:22px;margin:0 0 16px">` + title + `</h1>` + body +
		`<p style="font-size:12px;color:#7b8794;margin-top:32px">This is a demo message sent to an example.com address.</p>` +
		`</td></tr></table></td></tr></table></body></html>`
}

func demoMails() []mail {
	return []mail{
		{
			project: "shop", from: "Example Shop <orders@shop.example.com>", to: []string{"Alex Morgan <alex.morgan@example.com>"},
			subject: "Your order ORD-2026-11-48213 is confirmed",
			text: "Hi Alex,\n\nThanks for your order ORD-2026-11-48213.\n\n" +
				"1 x Trail Runner X (size 42)   89.90 EUR\n1 x Merino Running Socks       14.00 EUR\n1 x Rain Shell Jacket          16.00 EUR\n" +
				"Shipping (express)            10.00 EUR\nTotal                        129.90 EUR\n\n" +
				"Track your parcel: https://shop.example.com/orders/48213\n\nExample Shop",
			html: emailLayout("Thanks for your order, Alex!",
				`<p>Order <strong>ORD-2026-11-48213</strong> is confirmed and will ship tomorrow.</p>`+
					`<table width="100%" style="border-collapse:collapse;font-size:14px">`+
					`<tr><td style="padding:6px 0">Trail Runner X (size 42)</td><td align="right">89.90 EUR</td></tr>`+
					`<tr><td style="padding:6px 0">Merino Running Socks</td><td align="right">14.00 EUR</td></tr>`+
					`<tr><td style="padding:6px 0">Rain Shell Jacket</td><td align="right">16.00 EUR</td></tr>`+
					`<tr><td style="padding:6px 0">Shipping (express)</td><td align="right">10.00 EUR</td></tr>`+
					`<tr><td style="padding:6px 0;border-top:1px solid #e4e7eb"><strong>Total</strong></td><td align="right" style="border-top:1px solid #e4e7eb"><strong>129.90 EUR</strong></td></tr></table>`+
					`<p><a href="https://shop.example.com/orders/48213" style="display:inline-block;background:#2563eb;color:#fff;padding:10px 18px;border-radius:6px;text-decoration:none">Track your order</a></p>`),
		},
		{
			project: "shop", from: "Example Shop <no-reply@shop.example.com>", to: []string{"jamie.chen@example.com"},
			subject: "Reset your password",
			text:    "Hi Jamie,\n\nSomeone asked to reset the password for your Example Shop account.\nUse this link within 60 minutes: https://shop.example.com/password/reset/" + randHex(16) + "\n\nYour code: 482913\n\nIf it was not you, ignore this email.",
			html: emailLayout("Reset your password",
				`<p>Someone asked to reset the password for your account. The link works for 60 minutes.</p>`+
					`<p><a href="https://shop.example.com/password/reset/`+randHex(16)+`" style="display:inline-block;background:#2563eb;color:#fff;padding:10px 18px;border-radius:6px;text-decoration:none">Choose a new password</a></p>`+
					`<p>Or enter this code: <strong style="font-size:20px;letter-spacing:4px">482913</strong></p>`+
					`<p style="color:#7b8794">If it was not you, ignore this email.</p>`),
		},
		{
			project: "billing", from: "Example Billing <invoices@billing.example.com>", to: []string{"Finance Team <finance@example.com>"},
			cc:      []string{"Morgan Lee <morgan.lee@example.com>"},
			subject: "Invoice INV-2026-0043 for October 2026",
			text:    "Hello,\n\nPlease find attached invoice INV-2026-0043 for October 2026.\nAmount due: 1,290.00 EUR, due on 2 November 2026.\n\nPay online: https://billing.example.com/pay/INV-2026-0043\n\nExample Billing",
			html: emailLayout("Invoice INV-2026-0043",
				`<p>Please find attached the invoice for <strong>October 2026</strong>.</p>`+
					`<p style="font-size:28px;margin:8px 0"><strong>1,290.00 EUR</strong></p><p>Due on 2 November 2026.</p>`+
					`<p><a href="https://billing.example.com/pay/INV-2026-0043">Pay online</a></p>`),
			attach: []attachment{
				{name: "INV-2026-0043.pdf", contentType: "application/pdf", data: demoPDF("Invoice INV-2026-0043 - 1,290.00 EUR")},
				{name: "usage-2026-10.csv", contentType: "text/csv", data: []byte("date,seats,api_calls\n2026-10-01,12,48211\n2026-10-15,12,51904\n2026-10-31,14,57330\n")},
			},
		},
		{
			project: "billing", from: "Example Billing <billing@billing.example.com>", to: []string{"Taylor Brooks <taylor.brooks@example.com>"},
			bcc: []string{"audit@billing.example.com"}, priority: "1 (Highest)",
			subject: "Action needed: payment for INV-2026-0042 failed",
			text:    "Hi Taylor,\n\nWe could not charge your card for invoice INV-2026-0042 (49.00 EUR): insufficient funds.\nWe will try again on 10 November. Update your card: https://billing.example.com/payment-methods\n\nExample Billing",
			html: emailLayout("Your payment failed",
				`<p>We could not charge your card for invoice <strong>INV-2026-0042</strong> (49.00 EUR).</p>`+
					`<p style="background:#fff4e5;border-left:4px solid #f59e0b;padding:10px 12px">Reason: insufficient funds. Next attempt: 10 November.</p>`+
					`<p><a href="https://billing.example.com/payment-methods" style="display:inline-block;background:#dc2626;color:#fff;padding:10px 18px;border-radius:6px;text-decoration:none">Update payment method</a></p>`),
		},
	}
}

// --- Monolog ----------------------------------------------------------------------

var monologLevels = map[string]int{"DEBUG": 100, "INFO": 200, "NOTICE": 250, "WARNING": 300, "ERROR": 400, "CRITICAL": 500, "ALERT": 550}

func (s *seeder) seedMonolog() {
	if s.monolog == "" {
		return
	}
	conn, err := net.DialTimeout("tcp", s.monolog, 5*time.Second)
	if err != nil {
		s.fail("monolog connect", err)
		return
	}
	defer conn.Close()

	records := []struct {
		minutes                 int
		project, channel, level string
		message                 string
		context, extra          map[string]any
	}{
		{44, "shop", "orders", "INFO", "Order ORD-2026-11-48213 placed",
			map[string]any{"order_id": "ORD-2026-11-48213", "customer_id": 1042, "total": 129.90, "currency": "EUR", "items": 3},
			map[string]any{"url": "/checkout", "http_method": "POST", "ip": "203.0.113.24", "uid": randHex(4)}},
		{37, "shop", "inventory", "WARNING", "Low stock for SKU-1042 \"Trail Runner X\" size 42",
			map[string]any{"sku": "SKU-1042", "variant": "42", "available": 2, "threshold": 5, "warehouse": "berlin-1"},
			map[string]any{"memory_usage": "14 MB"}},
		{29, "shop", "security", "NOTICE", "Too many login attempts, account locked for 15 minutes",
			map[string]any{"email": "riley.novak@example.com", "attempts": 6, "ip": "198.51.100.88"}, nil},
		{18, "shop", "payments", "ERROR", "Charge request to billing-api failed: HTTP 503",
			map[string]any{"order_id": "ORD-2026-11-50117", "upstream": "https://billing.example.com/api/v1/charges", "status": 503, "retry_in": 30,
				"exception": map[string]any{"class": "Illuminate\\Http\\Client\\RequestException", "message": "HTTP request returned status code 503", "code": 503,
					"file": "/var/www/html/app/Payments/BillingClient.php:57"}},
			map[string]any{"url": "/checkout", "http_method": "POST"}},
		{6, "shop", "app", "DEBUG", "Cache warmed: 1,284 product prices",
			map[string]any{"duration_ms": 842, "keys": 1284}, nil},
		{41, "billing", "dunning", "INFO", "Dunning run finished",
			map[string]any{"reminders": 14, "suspended": 1, "duration_s": 3.4}, map[string]any{"command": "billing:send-reminders"}},
		{25, "billing", "invoices", "WARNING", "Tax provider slow: 7.8 s for INV-2026-0057",
			map[string]any{"invoice": "INV-2026-0057", "duration_ms": 7800, "timeout_ms": 10000}, nil},
		{9, "billing", "ledger", "CRITICAL", "Ledger balance mismatch for account acct_3345",
			map[string]any{"account": "acct_3345", "expected": "120.00", "actual": "118.50", "currency": "EUR"},
			map[string]any{"job": "App\\Jobs\\ReconcileLedger", "queue": "ledger"}},
		{3, "billing", "payments", "ERROR", "Card declined for INV-2026-0042",
			map[string]any{"invoice": "INV-2026-0042", "decline_code": "insufficient_funds", "attempt": 2}, nil},
	}

	for _, r := range records {
		ctx := map[string]any{"project": r.project}
		for k, v := range r.context {
			ctx[k] = v
		}
		extra := map[string]any{}
		for k, v := range r.extra {
			extra[k] = v
		}
		extra["hostname"] = r.project + "-web-1"
		line, _ := json.Marshal(map[string]any{
			"message":    r.message,
			"context":    ctx,
			"level":      monologLevels[r.level],
			"level_name": r.level,
			"channel":    r.channel,
			"datetime":   s.ago(r.minutes).Format("2006-01-02T15:04:05.000000-07:00"),
			"extra":      extra,
		})
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if _, err := conn.Write(append(line, '\n')); err != nil {
			s.fail("monolog write", err)
			return
		}
		s.count("monolog", 1)
	}
}
