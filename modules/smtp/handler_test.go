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

func TestConvertToUTF8(t *testing.T) {
	t.Run("utf-8 passthrough", func(t *testing.T) {
		input := []byte("Привет мир")
		got := convertToUTF8(input, "utf-8")
		if string(got) != "Привет мир" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("empty charset passthrough", func(t *testing.T) {
		input := []byte("hello")
		got := convertToUTF8(input, "")
		if string(got) != "hello" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("windows-1250 czech", func(t *testing.T) {
		// "týmu se vám ozve" in windows-1250
		win1250 := []byte{0x74, 0xFD, 0x6D, 0x75, 0x20, 0x73, 0x65, 0x20, 0x76, 0xE1, 0x6D, 0x20, 0x6F, 0x7A, 0x76, 0x65}
		got := convertToUTF8(win1250, "windows-1250")
		want := "týmu se vám ozve"
		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("iso-8859-1 latin", func(t *testing.T) {
		// "café" in iso-8859-1: 0x63 0x61 0x66 0xe9
		latin1 := []byte{0x63, 0x61, 0x66, 0xe9}
		got := convertToUTF8(latin1, "iso-8859-1")
		if string(got) != "café" {
			t.Errorf("got %q, want %q", got, "café")
		}
	})

	t.Run("unknown charset passthrough", func(t *testing.T) {
		input := []byte("data")
		got := convertToUTF8(input, "x-nonexistent-999")
		if string(got) != "data" {
			t.Errorf("got %q", got)
		}
	})
}

func TestParseEmail_CharsetConversion(t *testing.T) {
	t.Run("single part windows-1250", func(t *testing.T) {
		// "café" in windows-1250: 0x63 0x61 0x66 0xe9
		body := string([]byte{0x63, 0x61, 0x66, 0xe9})
		raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\nContent-Type: text/plain; charset=windows-1250\r\n\r\n" + body)

		parsed, _, err := parseEmail(raw, []string{"recipient@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Text != "café" {
			t.Errorf("Text = %q, want %q", parsed.Text, "café")
		}
	})

	t.Run("single part html iso-8859-2", func(t *testing.T) {
		// "Vaše zpráva" in iso-8859-2: V=56 a=61 š=B9 e=65 sp=20 z=7A p=70 r=72 á=E1 v=76 a=61
		body := string([]byte{0x56, 0x61, 0xB9, 0x65, 0x20, 0x7A, 0x70, 0x72, 0xE1, 0x76, 0x61})
		raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\nContent-Type: text/html; charset=iso-8859-2\r\n\r\n" + body)

		parsed, _, err := parseEmail(raw, []string{"recipient@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		want := "Vaše zpráva"
		if parsed.HTML != want {
			t.Errorf("HTML = %q, want %q", parsed.HTML, want)
		}
	})

	t.Run("multipart with charset", func(t *testing.T) {
		// "café" in iso-8859-1
		body := string([]byte{0x63, 0x61, 0x66, 0xe9})
		raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\nContent-Type: multipart/alternative; boundary=\"bnd\"\r\n\r\n--bnd\r\nContent-Type: text/plain; charset=iso-8859-1\r\n\r\n" + body + "\r\n--bnd\r\nContent-Type: text/html; charset=iso-8859-1\r\n\r\n<p>" + body + "</p>\r\n--bnd--")

		parsed, _, err := parseEmail(raw, []string{"recipient@example.com"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(parsed.Text, "café") {
			t.Errorf("Text = %q, want to contain %q", parsed.Text, "café")
		}
		if !strings.Contains(parsed.HTML, "café") {
			t.Errorf("HTML = %q, want to contain %q", parsed.HTML, "café")
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

func TestParseEmail_RFC2047Subject(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    string
	}{
		{
			name:    "Q-encoded UTF-8",
			subject: "=?utf-8?Q?Handled_with_Care_by_Chuck_Norris=E2=80=94Your_Package?=",
			want:    "Handled with Care by Chuck Norris\u2014Your Package",
		},
		{
			name:    "B-encoded UTF-8",
			subject: "=?utf-8?B?SGVsbG8gV29ybGQ=?=",
			want:    "Hello World",
		},
		{
			name:    "mixed encoded and plain",
			subject: "Handled with Care by Chuck =?utf-8?Q?Norris=E2=80=94Your?= Package Is Shipped!",
			want:    "Handled with Care by Chuck Norris\u2014Your Package Is Shipped!",
		},
		{
			name:    "plain subject unchanged",
			subject: "Plain Subject",
			want:    "Plain Subject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: " + tt.subject + "\r\nContent-Type: text/plain\r\n\r\ntest body")
			parsed, _, err := parseEmail(raw, []string{"recipient@example.com"})
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Subject != tt.want {
				t.Errorf("Subject = %q, want %q", parsed.Subject, tt.want)
			}
		})
	}
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
