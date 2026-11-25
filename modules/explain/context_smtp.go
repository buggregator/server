package explain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"regexp"
	"strings"
)

// Sizes of a mail's text for the agents.
const (
	mailBodyBytes   = 4000 // the text part, or the HTML read as text
	mailPartBytes   = 600  // a delivery report's own text
	mailAttachments = 20   // attachments listed by name
)

// mailHeaders are the headers worth an agent's look, beyond the addresses.
var mailHeaders = []string{"Date", "Message-ID", "In-Reply-To", "References", "List-Id", "List-Unsubscribe", "List-Unsubscribe-Post", "Auto-Submitted", "X-Mailer"}

type mailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (a mailAddress) String() string {
	if a.Name == "" {
		return a.Email
	}
	return a.Name + " <" + a.Email + ">"
}

// smtpBody is a mail as an agent reads it: who, what, the text once (the
// payload has it up to three times: text/html, parts, raw), the invite and
// the attachments by name only.
func smtpBody(payload json.RawMessage) string {
	var m struct {
		Subject string        `json:"subject"`
		From    []mailAddress `json:"from"`
		To      []mailAddress `json:"to"`
		Cc      []mailAddress `json:"cc"`
		Bcc     []string      `json:"bcc"`
		ReplyTo []mailAddress `json:"reply_to"`
		Text    string        `json:"text"`
		HTML    string        `json:"html"`
		Raw     string        `json:"raw"`
		Parts   []struct {
			Type      string `json:"type"`
			Label     string `json:"label"`
			Content   string `json:"content"`
			Size      int    `json:"size"`
			Truncated bool   `json:"truncated"`
		} `json:"parts"`
		Calendar *struct {
			Method      string `json:"method"`
			Summary     string `json:"summary"`
			Start       string `json:"start"`
			End         string `json:"end"`
			TZ          string `json:"tz"`
			AllDay      bool   `json:"all_day"`
			Location    string `json:"location"`
			Description string `json:"description"`
			URL         string `json:"url"`
			Status      string `json:"status"`
			Recurrence  string `json:"rrule"`
			Organizer   *struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"organizer"`
			Attendees []struct {
				Name   string `json:"name"`
				Email  string `json:"email"`
				Status string `json:"status"`
				Role   string `json:"role"`
			} `json:"attendees"`
		} `json:"calendar"`
	}
	if json.Unmarshal(payload, &m) != nil {
		return "Mail (JSON):\n" + cut(compact(payload), maxEventBytes-200) + "\n"
	}

	head, atts := mailRaw(m.Raw)
	var b strings.Builder
	writeField(&b, "Subject", m.Subject)
	writeAddresses(&b, "From", m.From)
	writeAddresses(&b, "Reply-To", m.ReplyTo)
	writeAddresses(&b, "To", m.To)
	writeAddresses(&b, "Cc", m.Cc)
	writeField(&b, "Bcc", strings.Join(m.Bcc, ", "))
	for _, h := range mailHeaders {
		writeField(&b, h, head.Get(h))
	}

	if c := m.Calendar; c != nil {
		b.WriteString("\nCalendar invite")
		if c.Method != "" {
			b.WriteString(" (" + c.Method + ")")
		}
		b.WriteString(":\n")
		writeField(&b, "  Summary", c.Summary)
		when := strings.TrimSpace(c.Start + " – " + c.End)
		if c.End == "" {
			when = c.Start
		}
		if c.AllDay {
			when += " (all day)"
		}
		writeField(&b, "  When", strings.TrimSpace(when+" "+c.TZ))
		writeField(&b, "  Repeats", c.Recurrence)
		writeField(&b, "  Status", c.Status)
		writeField(&b, "  Location", c.Location)
		writeField(&b, "  URL", c.URL)
		if o := c.Organizer; o != nil {
			writeField(&b, "  Organizer", mailAddress{Email: o.Email, Name: o.Name}.String())
		}
		for i, a := range c.Attendees {
			if i == topRows {
				fmt.Fprintf(&b, "  …and %d more attendees\n", len(c.Attendees)-topRows)
				break
			}
			line := mailAddress{Email: a.Email, Name: a.Name}.String()
			if s := strings.Join(nonEmpty(a.Role, a.Status), ", "); s != "" {
				line += " (" + s + ")"
			}
			writeField(&b, "  Attendee", line)
		}
		if d := strings.TrimSpace(c.Description); d != "" {
			b.WriteString("  Description: " + cut(d, 500) + "\n")
		}
	}

	body, from := strings.TrimSpace(m.Text), "text part"
	if body == "" && strings.TrimSpace(m.HTML) != "" {
		body, from = htmlText(m.HTML), "HTML part read as text"
	}
	if body != "" {
		b.WriteString("\nBody (" + from + "):\n" + cut(body, mailBodyBytes) + "\n")
	} else {
		b.WriteString("\nBody: empty\n")
	}
	if m.Text != "" && m.HTML != "" {
		fmt.Fprintf(&b, "The mail has an HTML part too (%d bytes).\n", len(m.HTML))
	}

	if len(m.Parts) > 0 {
		b.WriteString("\nOther body parts:\n")
		for _, p := range m.Parts {
			name := p.Type
			if p.Label != "" {
				name = p.Label + " (" + p.Type + ")"
			}
			fmt.Fprintf(&b, "  %s, %d bytes\n", oneLine(name, 200), p.Size)
			// A bounce's reason is in its delivery report.
			if strings.HasPrefix(p.Type, "message/delivery-status") || strings.HasPrefix(p.Type, "message/disposition-notification") {
				b.WriteString(indent(cut(strings.TrimSpace(p.Content), mailPartBytes), "    ") + "\n")
			}
		}
	}

	if len(atts) > 0 {
		fmt.Fprintf(&b, "\nAttachments (%d, contents not included):\n", len(atts))
		for i, a := range atts {
			if i == mailAttachments {
				fmt.Fprintf(&b, "  …and %d more\n", len(atts)-mailAttachments)
				break
			}
			fmt.Fprintf(&b, "  %s (%s, %s)\n", oneLine(a.name, 200), a.mime, size(a.size))
		}
	}
	return b.String()
}

func writeAddresses(b *strings.Builder, name string, list []mailAddress) {
	s := make([]string, 0, len(list))
	for _, a := range list {
		s = append(s, a.String())
	}
	writeField(b, name, strings.Join(s, ", "))
}

type mailAttachment struct {
	name, mime string
	size       int
}

// mailRaw reads the headers and the attachment list from the raw mail; the
// payload keeps neither. Attachments are told the way the smtp module tells
// them: a part with an attachment or inline disposition.
func mailRaw(raw string) (mail.Header, []mailAttachment) {
	msg, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		return mail.Header{}, nil
	}
	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if !strings.HasPrefix(mediaType, "multipart/") || params["boundary"] == "" {
		return msg.Header, nil
	}
	var atts []mailAttachment
	walkParts(multipart.NewReader(msg.Body, params["boundary"]), &atts, 0)
	return msg.Header, atts
}

func walkParts(mr *multipart.Reader, atts *[]mailAttachment, depth int) {
	if depth > 8 {
		return
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			return
		}
		mediaType, params, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if strings.HasPrefix(mediaType, "multipart/") {
			if params["boundary"] != "" {
				walkParts(multipart.NewReader(part, params["boundary"]), atts, depth+1)
			}
			continue
		}
		disp := part.Header.Get("Content-Disposition")
		if !strings.HasPrefix(disp, "attachment") && !strings.HasPrefix(disp, "inline") {
			continue
		}
		content, _ := io.ReadAll(part)
		n := len(content)
		if strings.EqualFold(part.Header.Get("Content-Transfer-Encoding"), "base64") {
			clean := strings.Map(func(r rune) rune {
				if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
					return -1
				}
				return r
			}, string(content))
			if d, err := base64.StdEncoding.DecodeString(clean); err == nil {
				n = len(d)
			}
		}
		name := part.FileName()
		if d, err := new(mime.WordDecoder).DecodeHeader(name); err == nil {
			name = d
		}
		if name == "" {
			name = "unnamed"
		}
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		*atts = append(*atts, mailAttachment{name: name, mime: mediaType, size: n})
	}
}

var (
	htmlDrop   = regexp.MustCompile(`(?is)<(head|style|script|title)\b.*?</(head|style|script|title)\s*>|<!--.*?-->`)
	htmlBreak  = regexp.MustCompile(`(?i)<br\s*/?>|</(p|div|tr|li|h[1-6]|table|blockquote)\s*>`)
	htmlCell   = regexp.MustCompile(`(?i)</t[dh]\s*>`)
	blankLines = regexp.MustCompile(`\n{3,}`)
)

// htmlText reads an HTML mail as plain text: no head, styles or scripts,
// a line break per block, entities decoded.
func htmlText(s string) string {
	s = htmlDrop.ReplaceAllString(s, "")
	s = htmlBreak.ReplaceAllString(s, "\n")
	s = htmlCell.ReplaceAllString(s, " ")
	s = html.UnescapeString(htmlTag.ReplaceAllString(s, ""))
	lines := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	for i, l := range lines {
		lines[i] = strings.Join(strings.Fields(strings.ReplaceAll(l, " ", " ")), " ")
	}
	return strings.TrimSpace(blankLines.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

func indent(s, pad string) string {
	return pad + strings.ReplaceAll(s, "\n", "\n"+pad)
}

func size(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	return kb(int64(n))
}
