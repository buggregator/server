package smtp

import (
	"strings"
	"time"
)

// maxPartContent caps the stored text of one alternative part; the whole
// message stays in Raw.
const maxPartContent = 1 << 20

// EmailPart is a body part other than the text/plain and text/html bodies:
// the Apple Watch version, an AMP version, a calendar invite, a delivery
// report. Before parts existed these were dropped.
type EmailPart struct {
	Type      string `json:"type"`  // media type, e.g. text/watch-html
	Label     string `json:"label"` // short name for a tab: "Apple Watch", "AMP"…
	Content   string `json:"content"`
	Size      int    `json:"size"`                // bytes before truncation
	Truncated bool   `json:"truncated,omitempty"` // Content cut at 1 MB
}

// partLabels names the known alternative parts.
var partLabels = map[string]string{
	"text/watch-html":                  "Apple Watch",
	"text/x-amp-html":                  "AMP",
	"text/calendar":                    "Calendar",
	"application/ics":                  "Calendar",
	"text/enriched":                    "Enriched text",
	"text/richtext":                    "Rich text",
	"text/markdown":                    "Markdown",
	"text/x-markdown":                  "Markdown",
	"message/delivery-status":          "Delivery status",
	"message/global-delivery-status":   "Delivery status",
	"message/disposition-notification": "Read receipt",
	"text/rfc822-headers":              "Original headers",
}

// keepAsPart reports whether a non-attachment body part of this media type
// is kept as an EmailPart: any text/* except the plain and HTML bodies, the
// calendar types and the delivery / read reports.
func keepAsPart(mediaType string) bool {
	if _, ok := partLabels[mediaType]; ok {
		return true
	}
	return strings.HasPrefix(mediaType, "text/")
}

func newPart(mediaType string, content []byte) EmailPart {
	p := EmailPart{Type: mediaType, Label: partLabels[mediaType], Size: len(content)}
	if p.Label == "" {
		p.Label = mediaType
	}
	if len(content) > maxPartContent {
		content, p.Truncated = content[:maxPartContent], true
	}
	p.Content = string(content)
	return p
}

// isCalendar reports whether a part or attachment carries iCalendar data.
func isCalendar(mediaType, filename string) bool {
	return mediaType == "text/calendar" || mediaType == "application/ics" ||
		strings.HasSuffix(strings.ToLower(filename), ".ics")
}

// CalendarInvite is the first event of an iCalendar part or .ics attachment,
// enough for an invite card. Times are RFC 3339 when the time zone is known
// (UTC or a TZID this server knows); otherwise local "2006-01-02T15:04:05"
// with TZ naming the zone, or a date for all-day events.
type CalendarInvite struct {
	Method      string             `json:"method,omitempty"` // REQUEST, CANCEL, REPLY…
	Summary     string             `json:"summary"`
	Start       string             `json:"start,omitempty"`
	End         string             `json:"end,omitempty"`
	TZ          string             `json:"tz,omitempty"`
	AllDay      bool               `json:"all_day,omitempty"`
	Location    string             `json:"location,omitempty"`
	Description string             `json:"description,omitempty"`
	URL         string             `json:"url,omitempty"`
	Status      string             `json:"status,omitempty"` // CONFIRMED, TENTATIVE, CANCELLED
	UID         string             `json:"uid,omitempty"`
	Sequence    string             `json:"sequence,omitempty"`
	Recurrence  string             `json:"rrule,omitempty"`
	Organizer   *CalendarPerson    `json:"organizer,omitempty"`
	Attendees   []CalendarAttendee `json:"attendees,omitempty"`
}

// CalendarPerson is an organizer: name and email.
type CalendarPerson struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// CalendarAttendee is an invited person with their answer.
type CalendarAttendee struct {
	CalendarPerson
	Status string `json:"status,omitempty"` // PARTSTAT: NEEDS-ACTION, ACCEPTED, DECLINED, TENTATIVE
	Role   string `json:"role,omitempty"`   // REQ-PARTICIPANT, OPT-PARTICIPANT, CHAIR
}

type icsLine struct {
	name   string
	params map[string]string
	value  string
}

// parseICSLine splits `NAME;P1=V1;P2="V 2":value`.
func parseICSLine(line string) (icsLine, bool) {
	// The value starts at the first ':' outside double quotes.
	inQuote, colon := false, -1
	for i, r := range line {
		if r == '"' {
			inQuote = !inQuote
		} else if r == ':' && !inQuote {
			colon = i
			break
		}
	}
	if colon < 0 {
		return icsLine{}, false
	}
	head, value := line[:colon], line[colon+1:]
	l := icsLine{value: value, params: map[string]string{}}
	parts := splitOutsideQuotes(head, ';')
	l.name = strings.ToUpper(parts[0])
	for _, p := range parts[1:] {
		if k, v, ok := strings.Cut(p, "="); ok {
			l.params[strings.ToUpper(k)] = strings.Trim(v, `"`)
		}
	}
	return l, true
}

func splitOutsideQuotes(s string, sep rune) []string {
	var out []string
	inQuote, start := false, 0
	for i, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case r == sep && !inQuote:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// icsText undoes iCalendar text escaping (\n, \, \; \\).
func icsText(s string) string {
	r := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return r.Replace(s)
}

func icsPerson(l icsLine) CalendarPerson {
	email := l.value
	if len(email) >= 7 && strings.EqualFold(email[:7], "mailto:") {
		email = email[7:]
	}
	return CalendarPerson{Name: l.params["CN"], Email: email}
}

// icsTime converts a DTSTART/DTEND value: RFC 3339 when the zone is known,
// else local time and the zone's name; a date for VALUE=DATE.
func icsTime(l icsLine) (value, tz string, allDay bool) {
	v := strings.TrimSpace(l.value)
	if l.params["VALUE"] == "DATE" || len(v) == 8 {
		if t, err := time.Parse("20060102", v); err == nil {
			return t.Format("2006-01-02"), "", true
		}
		return v, "", true
	}
	if strings.HasSuffix(v, "Z") {
		if t, err := time.Parse("20060102T150405Z", v); err == nil {
			return t.UTC().Format(time.RFC3339), "UTC", false
		}
		return v, "", false
	}
	tzid := l.params["TZID"]
	if tzid != "" {
		if loc, err := time.LoadLocation(tzid); err == nil {
			if t, err := time.ParseInLocation("20060102T150405", v, loc); err == nil {
				return t.Format(time.RFC3339), tzid, false
			}
		}
	}
	if t, err := time.Parse("20060102T150405", v); err == nil {
		return t.Format("2006-01-02T15:04:05"), tzid, false
	}
	return v, tzid, false
}

// parseCalendar reads the first VEVENT of an iCalendar text; nil when there
// is none.
func parseCalendar(data string) *CalendarInvite {
	// Unfold: a line starting with a space or tab continues the previous one.
	data = strings.ReplaceAll(data, "\r\n", "\n")
	data = strings.ReplaceAll(data, "\n ", "")
	data = strings.ReplaceAll(data, "\n\t", "")

	var inv CalendarInvite
	method, inEvent, found, depth := "", false, false, 0
	for _, raw := range strings.Split(data, "\n") {
		l, ok := parseICSLine(strings.TrimRight(raw, "\r"))
		if !ok {
			continue
		}
		switch l.name {
		case "METHOD":
			method = strings.ToUpper(strings.TrimSpace(l.value))
			continue
		case "BEGIN":
			if strings.EqualFold(l.value, "VEVENT") && !found {
				inEvent, found = true, true
			} else if inEvent {
				depth++ // VALARM and the like inside the event
			}
			continue
		case "END":
			if inEvent && depth > 0 {
				depth--
			} else if strings.EqualFold(l.value, "VEVENT") {
				inEvent = false
			}
			continue
		}
		if !inEvent || depth > 0 {
			continue
		}
		switch l.name {
		case "SUMMARY":
			inv.Summary = icsText(l.value)
		case "DTSTART":
			inv.Start, inv.TZ, inv.AllDay = icsTime(l)
		case "DTEND":
			inv.End, _, _ = icsTime(l)
		case "LOCATION":
			inv.Location = icsText(l.value)
		case "DESCRIPTION":
			inv.Description = truncateRunes(icsText(l.value), 4000)
		case "URL":
			inv.URL = l.value
		case "STATUS":
			inv.Status = strings.ToUpper(l.value)
		case "UID":
			inv.UID = l.value
		case "SEQUENCE":
			inv.Sequence = l.value
		case "RRULE":
			inv.Recurrence = l.value
		case "ORGANIZER":
			p := icsPerson(l)
			inv.Organizer = &p
		case "ATTENDEE":
			if len(inv.Attendees) < 200 {
				inv.Attendees = append(inv.Attendees, CalendarAttendee{
					CalendarPerson: icsPerson(l),
					Status:         strings.ToUpper(l.params["PARTSTAT"]),
					Role:           strings.ToUpper(l.params["ROLE"]),
				})
			}
		}
	}
	if !found {
		return nil
	}
	inv.Method = method
	return &inv
}

func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
