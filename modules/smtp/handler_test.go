package smtp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseEmail_PlainText(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test Subject\r\nContent-Type: text/plain\r\nMessage-ID: <test123@example.com>\r\n\r\nHello, this is a test email.")

	parsed, atts, err := parseEmail(raw, []string{"recipient@example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if parsed.Subject != "Test Subject" {
		t.Errorf("Subject = %q, want %q", parsed.Subject, "Test Subject")
	}
	if len(parsed.From) != 1 || parsed.From[0].Email != "sender@example.com" {
		t.Errorf("From = %+v", parsed.From)
	}
	if len(parsed.To) != 1 || parsed.To[0].Email != "recipient@example.com" {
		t.Errorf("To = %+v", parsed.To)
	}
	if parsed.Text != "Hello, this is a test email." {
		t.Errorf("Text = %q", parsed.Text)
	}
	if parsed.HTML != "" {
		t.Errorf("HTML should be empty, got %q", parsed.HTML)
	}
	if parsed.ID == nil || *parsed.ID != "<test123@example.com>" {
		t.Errorf("ID = %v", parsed.ID)
	}
	if len(atts) != 0 {
		t.Errorf("expected no attachments, got %d", len(atts))
	}
}

func TestParseEmail_HTML(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: HTML Test\r\nContent-Type: text/html\r\n\r\n<p>Hello HTML</p>")

	parsed, _, err := parseEmail(raw, []string{"recipient@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if parsed.HTML != "<p>Hello HTML</p>" {
		t.Errorf("HTML = %q", parsed.HTML)
	}
	if parsed.Text != "" {
		t.Errorf("Text should be empty")
	}
}

func TestParseEmail_Multipart(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Multipart Test\r\nContent-Type: multipart/alternative; boundary=\"boundary123\"\r\nMessage-ID: <multi@example.com>\r\n\r\n--boundary123\r\nContent-Type: text/plain\r\n\r\nPlain text body\r\n--boundary123\r\nContent-Type: text/html\r\n\r\n<p>HTML body</p>\r\n--boundary123--")

	parsed, _, err := parseEmail(raw, []string{"recipient@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(parsed.Text, "Plain text body") {
		t.Errorf("Text = %q", parsed.Text)
	}
	if !strings.Contains(parsed.HTML, "HTML body") {
		t.Errorf("HTML = %q", parsed.HTML)
	}
}

func TestParseEmail_BCC(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: visible@example.com\r\nSubject: BCC Test\r\nContent-Type: text/plain\r\n\r\ntest")

	recipients := []string{"visible@example.com", "hidden@example.com"}
	parsed, _, err := parseEmail(raw, recipients)
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Bcc) != 1 || parsed.Bcc[0] != "hidden@example.com" {
		t.Errorf("Bcc = %v, want [hidden@example.com]", parsed.Bcc)
	}
}

func TestParseEmail_NoBCC(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nTo: visible@example.com\r\nSubject: No BCC\r\nContent-Type: text/plain\r\n\r\ntest")

	parsed, _, err := parseEmail(raw, []string{"visible@example.com"})
	if err != nil {
		t.Fatal(err)
	}

	if len(parsed.Bcc) != 0 {
		t.Errorf("Bcc = %v, want empty", parsed.Bcc)
	}
}

func TestDecodeContent(t *testing.T) {
	t.Run("base64", func(t *testing.T) {
		encoded := "SGVsbG8gV29ybGQ=" // "Hello World"
		got := decodeContent([]byte(encoded), "base64")
		if string(got) != "Hello World" {
			t.Errorf("got %q, want %q", got, "Hello World")
		}
	})

	t.Run("quoted-printable", func(t *testing.T) {
		encoded := "Hello=20World"
		got := decodeContent([]byte(encoded), "quoted-printable")
		if string(got) != "Hello World" {
			t.Errorf("got %q, want %q", got, "Hello World")
		}
	})

	t.Run("no encoding", func(t *testing.T) {
		data := []byte("plain text")
		got := decodeContent(data, "")
		if string(got) != "plain text" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("7bit passthrough", func(t *testing.T) {
		data := []byte("ascii text")
		got := decodeContent(data, "7bit")
		if string(got) != "ascii text" {
			t.Errorf("got %q", got)
		}
	})
}

func TestParseAddresses(t *testing.T) {
	t.Run("name and email", func(t *testing.T) {
		header := make(map[string][]string)
		header["From"] = []string{"John Doe <john@example.com>"}
		addrs := parseAddresses(header, "From")
		if len(addrs) != 1 {
			t.Fatalf("len = %d", len(addrs))
		}
		if addrs[0].Email != "john@example.com" {
			t.Errorf("Email = %q", addrs[0].Email)
		}
		if addrs[0].Name != "John Doe" {
			t.Errorf("Name = %q", addrs[0].Name)
		}
	})

	t.Run("email only", func(t *testing.T) {
		header := make(map[string][]string)
		header["To"] = []string{"user@example.com"}
		addrs := parseAddresses(header, "To")
		if len(addrs) != 1 {
			t.Fatalf("len = %d", len(addrs))
		}
		if addrs[0].Email != "user@example.com" {
			t.Errorf("Email = %q", addrs[0].Email)
		}
	})

	t.Run("empty header", func(t *testing.T) {
		header := make(map[string][]string)
		addrs := parseAddresses(header, "Missing")
		if len(addrs) != 0 {
			t.Errorf("expected empty, got %d", len(addrs))
		}
	})
}

func TestPreviewMapper(t *testing.T) {
	m := &previewMapper{}

	t.Run("ToPreview extracts subject/from/to", func(t *testing.T) {
		email := ParsedEmail{
			Subject: "Test",
			From:    []EmailAddress{{Email: "from@test.com", Name: "Sender"}},
			To:      []EmailAddress{{Email: "to@test.com", Name: "Recipient"}},
			Text:    "body content that should be excluded",
			HTML:    "<p>html content that should be excluded</p>",
		}
		payload, _ := json.Marshal(email)
		preview, err := m.ToPreview(payload)
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]any
		json.Unmarshal(preview, &p)
		if p["subject"] != "Test" {
			t.Error("missing subject")
		}
		if _, ok := p["text"]; ok {
			t.Error("text should not be in preview")
		}
		if _, ok := p["html"]; ok {
			t.Error("html should not be in preview")
		}
	})

	t.Run("ToSearchableText", func(t *testing.T) {
		email := ParsedEmail{
			Subject: "Important",
			From:    []EmailAddress{{Email: "from@test.com", Name: "Sender"}},
			To:      []EmailAddress{{Email: "to@test.com", Name: "Recipient"}},
		}
		payload, _ := json.Marshal(email)
		text := m.ToSearchableText(payload)
		if !strings.Contains(text, "Important") {
			t.Error("expected subject in text")
		}
		if !strings.Contains(text, "from@test.com") {
			t.Error("expected from email in text")
		}
	})
}
