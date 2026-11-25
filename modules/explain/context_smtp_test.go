package explain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

func mailPayload(t *testing.T, fields map[string]any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func readEML(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile("../smtp/testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSmtpBody(t *testing.T) {
	// A real-size HTML mail: the payload carries the body in html and raw;
	// the agents get it once, as text.
	rows := strings.Repeat(`<tr><td style="padding:4px">Item&nbsp;#15</td><td>&euro;9.99</td></tr>`, 300)
	bigHTML := `<html><head><title>Receipt</title><style>td{color:red}</style></head><body><p>Hello&nbsp;Ann,</p><p>Your order &amp; receipt:</p><table>` + rows + `</table><script>track()</script></body></html>`
	bigRaw := "From: Shop <shop@example.com>\r\nTo: ann@example.com\r\nSubject: Receipt\r\nDate: Wed, 25 Nov 2026 10:00:00 +0000\r\nMessage-ID: <r-15@shop.test>\r\nList-Unsubscribe: <https://shop.test/u>\r\nContent-Type: multipart/mixed; boundary=\"m\"\r\n\r\n" +
		"--m\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" + bigHTML + "\r\n" +
		"--m\r\nContent-Type: application/pdf; name=\"receipt.pdf\"\r\nContent-Disposition: attachment; filename=\"=?utf-8?Q?r=C3=A9ceipt.pdf?=\"\r\nContent-Transfer-Encoding: base64\r\n\r\nSGVsbG8g\r\nd29ybGQh\r\n--m--\r\n"

	tests := []struct {
		name    string
		payload json.RawMessage
		want    []string
		gone    []string
	}{
		{"big html mail", mailPayload(t, map[string]any{
			"subject": "Receipt", "from": []map[string]string{{"email": "shop@example.com", "name": "Shop"}},
			"to": []map[string]string{{"email": "ann@example.com"}}, "cc": []any{}, "bcc": []string{"audit@example.com"},
			"text": "", "html": bigHTML, "raw": bigRaw,
		}), []string{
			"Subject: Receipt", "From: Shop <shop@example.com>", "To: ann@example.com", "Bcc: audit@example.com",
			"Date: Wed, 25 Nov 2026 10:00:00 +0000", "Message-ID: <r-15@shop.test>", "List-Unsubscribe: <https://shop.test/u>",
			"Body (HTML part read as text):\nHello Ann,", "Your order & receipt:", "Item #15 €9.99", "[cut to fit]",
			"Attachments (1, contents not included):\n  réceipt.pdf (application/pdf, 12 B)",
		}, []string{"<td", "color:red", "track()", "&nbsp;", "SGVsbG8g", "<title>"}},

		{"calendar invite", mailPayload(t, map[string]any{
			"subject": "Invitation: Release sync", "from": []map[string]string{{"email": "ann@example.com", "name": "Ann Lee"}},
			"to": []map[string]string{{"email": "bob@example.com"}}, "text": "Join us", "html": "<p>Join us</p>",
			"raw":   readEML(t, "google-invite.eml"),
			"parts": []map[string]any{{"type": "text/calendar", "label": "Calendar", "content": "BEGIN:VCALENDAR…", "size": 600}},
			"calendar": map[string]any{
				"method": "REQUEST", "summary": "Release sync", "start": "2026-12-01T09:00:00Z", "end": "2026-12-01T09:30:00Z",
				"location": "Room 1", "organizer": map[string]string{"name": "Ann Lee", "email": "ann@example.com"},
				"attendees": []map[string]string{{"email": "bob@example.com", "status": "NEEDS-ACTION", "role": "REQ-PARTICIPANT"}},
			},
		}), []string{
			"Calendar invite (REQUEST):", "  Summary: Release sync", "  When: 2026-12-01T09:00:00Z – 2026-12-01T09:30:00Z",
			"  Organizer: Ann Lee <ann@example.com>", "  Attendee: bob@example.com (REQ-PARTICIPANT, NEEDS-ACTION)",
			"Body (text part):\nJoin us", "The mail has an HTML part too", "  Calendar (text/calendar), 600 bytes",
			"  invite.ics (application/ics, ",
		}, []string{"BEGIN:VCALENDAR"}},

		{"bounce", mailPayload(t, map[string]any{
			"subject": "Undelivered Mail Returned to Sender", "text": "could not be delivered", "raw": readEML(t, "bounce.eml"),
			"parts": []map[string]any{{"type": "message/delivery-status", "label": "Delivery report", "content": "Status: 5.1.1\nDiagnostic-Code: smtp; 550 no such user", "size": 60}},
		}), []string{"Delivery report (message/delivery-status), 60 bytes", "    Diagnostic-Code: smtp; 550 no such user"}, []string{"Attachments"}},

		{"not a mail", json.RawMessage(`[1,2]`), []string{"Mail (JSON):", "[1,2]"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := eventBody(event.Event{Type: "smtp", Payload: tt.payload})
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(body, w) {
					t.Errorf("lacks %q:\n%s", w, body)
				}
			}
			for _, g := range tt.gone {
				if strings.Contains(body, g) {
					t.Errorf("has %q:\n%s", g, body)
				}
			}
			if len(body) > maxEventBytes {
				t.Errorf("body is %d bytes, over %d", len(body), maxEventBytes)
			}
		})
	}
}
