package smtp

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func parseFixture(t *testing.T, name string) (*ParsedEmail, []parsedAttachment) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	p, atts, err := parseEmail(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p, atts
}

func partTypes(p *ParsedEmail) string {
	var s []string
	for _, x := range p.Parts {
		s = append(s, x.Type+"="+x.Label)
	}
	return strings.Join(s, ",")
}

func TestParseEmail_AlternativeParts(t *testing.T) {
	cases := []struct {
		file, parts, text, html string
	}{
		{"watch.eml", "text/watch-html=Apple Watch", "Your order shipped.", "<h1>Your order shipped</h1>"},
		{"amp.eml", "text/x-amp-html=AMP", "Vote in our poll.", "<p>Vote in our poll.</p>"},
		{"google-invite.eml", "text/calendar=Calendar", "You have been invited", "<b>Release sync</b>"},
		{"outlook-cancel.eml", "text/calendar=Calendar", "", ""},
		{"ics-attachment.eml", "", "", "Booking confirmed."},
		{"bounce.eml", "message/delivery-status=Delivery status,text/rfc822-headers=Original headers", "could not be delivered", ""},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			p, _ := parseFixture(t, tc.file)
			if got := partTypes(p); got != tc.parts {
				t.Errorf("parts %q, want %q", got, tc.parts)
			}
			if !strings.Contains(p.Text, tc.text) || (tc.text == "" && p.Text != "") {
				t.Errorf("text %q, want %q", p.Text, tc.text)
			}
			if !strings.Contains(p.HTML, tc.html) || (tc.html == "" && p.HTML != "") {
				t.Errorf("html %q, want %q", p.HTML, tc.html)
			}
			// The alternative versions never leak into the main bodies.
			for _, x := range p.Parts {
				if strings.Contains(p.HTML, x.Content) || strings.Contains(p.Text, x.Content) {
					t.Errorf("%s content leaked into a main body", x.Type)
				}
			}
		})
	}

	p, _ := parseFixture(t, "watch.eml")
	if c := p.Parts[0].Content; c != "<b>Shipped</b> ✓" {
		t.Errorf("watch content not decoded: %q", c)
	}
	p, _ = parseFixture(t, "bounce.eml")
	if !strings.Contains(p.Parts[0].Content, "Status: 5.1.1") {
		t.Errorf("delivery status %q", p.Parts[0].Content)
	}
}

func TestParseEmail_CalendarInvite(t *testing.T) {
	p, atts := parseFixture(t, "google-invite.eml")
	if len(atts) != 1 || atts[0].Filename != "invite.ics" {
		t.Errorf("invite.ics attachment kept: %+v", atts)
	}
	c := p.Calendar
	if c == nil {
		t.Fatal("no calendar")
	}
	want := CalendarInvite{
		Method: "REQUEST", Summary: "Release sync", Start: "2026-12-01T09:00:00Z", End: "2026-12-01T09:30:00Z", TZ: "UTC",
		Location: "Room 4; floor 2", Description: "Agenda:\n1. Release, docs\n2. Q&A", Status: "CONFIRMED",
		UID: "abc123@google.com", Sequence: "0",
		Organizer: &CalendarPerson{Name: "Ann Lee", Email: "ann@example.com"},
		Attendees: []CalendarAttendee{
			{CalendarPerson{Name: "bob@example.com", Email: "bob@example.com"}, "NEEDS-ACTION", "REQ-PARTICIPANT"},
			{CalendarPerson{Name: "Lee, Ann", Email: "ann@example.com"}, "ACCEPTED", "REQ-PARTICIPANT"},
		},
	}
	got, _ := json.Marshal(c)
	exp, _ := json.Marshal(want)
	if string(got) != string(exp) {
		t.Errorf("invite\n got %s\nwant %s", got, exp)
	}

	// Outlook: a Windows zone name this server cannot load stays local time.
	p, _ = parseFixture(t, "outlook-cancel.eml")
	if c := p.Calendar; c == nil || c.Method != "CANCEL" || c.Status != "CANCELLED" || c.Summary != "Planning" ||
		c.Start != "2026-12-03T14:00:00" || c.TZ != "W. Europe Standard Time" || c.Organizer.Email != "bob@example.com" {
		t.Errorf("outlook cancel %+v", c)
	}

	// Only an .ics attachment (an app's booking mail): all-day event.
	p, _ = parseFixture(t, "ics-attachment.eml")
	if c := p.Calendar; c == nil || c.Summary != "Haircut" || !c.AllDay || c.Start != "2026-12-05" || c.End != "2026-12-06" {
		t.Errorf("ics attachment %+v", c)
	}

	p, _ = parseFixture(t, "watch.eml")
	if p.Calendar != nil {
		t.Error("calendar on a message without one")
	}
}

func TestICSTime(t *testing.T) {
	l, _ := parseICSLine("DTSTART;TZID=Europe/Berlin:20261205T100000")
	if v, tz, all := icsTime(l); v != "2026-12-05T10:00:00+01:00" || tz != "Europe/Berlin" || all {
		t.Errorf("berlin: %s %s %v", v, tz, all)
	}
	if parseCalendar("no calendar here") != nil {
		t.Error("calendar from garbage")
	}
}

func TestParseEmail_PartsOmittedAndTruncated(t *testing.T) {
	p, _, _ := parseEmail([]byte("From: a@b.c\r\nTo: d@e.f\r\nSubject: hi\r\n\r\nhello"), nil)
	b, _ := json.Marshal(p)
	if strings.Contains(string(b), `"parts"`) || strings.Contains(string(b), `"calendar"`) {
		t.Errorf("ordinary message carries parts/calendar: %s", b)
	}

	big := strings.Repeat("x", maxPartContent+10)
	raw := "From: a@b.c\r\nContent-Type: multipart/alternative; boundary=z\r\n\r\n--z\r\nContent-Type: text/watch-html\r\n\r\n" + big + "\r\n--z--\r\n"
	p, _, _ = parseEmail([]byte(raw), nil)
	if len(p.Parts) != 1 || !p.Parts[0].Truncated || len(p.Parts[0].Content) != maxPartContent || p.Parts[0].Size != len(big) {
		t.Errorf("big part: %d parts, truncated %v", len(p.Parts), len(p.Parts) == 1 && p.Parts[0].Truncated)
	}
}

func TestPreviewMapper_PartsAndInvite(t *testing.T) {
	m := &previewMapper{}
	p, _ := parseFixture(t, "google-invite.eml")
	payload, _ := json.Marshal(p)
	out, _ := m.ToPreview(payload)
	if !strings.Contains(string(out), `"parts":[{"label":"Calendar","type":"text/calendar"}]`) ||
		!strings.Contains(string(out), `"calendar":{"all_day":false,"method":"REQUEST","start":"2026-12-01T09:00:00Z","summary":"Release sync","tz":"UTC"}`) {
		t.Errorf("preview %s", out)
	}
	if !strings.Contains(m.ToSearchableText(payload), "Release sync") {
		t.Error("invite summary not searchable")
	}
	p, _ = parseFixture(t, "watch.eml")
	payload, _ = json.Marshal(p)
	if out, _ := m.ToPreview(payload); strings.Contains(string(out), "calendar") {
		t.Errorf("preview without invite: %s", out)
	}
}
