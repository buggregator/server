package smtp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
	gosmtp "github.com/emersion/go-smtp"
)

// smtpServer wraps go-smtp and implements tcp.Starter.
type smtpServer struct {
	server       *gosmtp.Server
	eventService EventStorer
}

func newSMTPServer(addr string, es EventStorer) *smtpServer {
	be := &backend{eventService: es}
	s := gosmtp.NewServer(be)
	s.Addr = addr
	s.Domain = "localhost"
	s.ReadTimeout = 60 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 10 * 1024 * 1024
	s.AllowInsecureAuth = true

	return &smtpServer{server: s, eventService: es}
}

func (s *smtpServer) Start(ctx context.Context) error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			slog.Error("SMTP server error", "err", err)
		}
	}()
	go func() {
		<-ctx.Done()
		s.server.Close()
	}()
	return nil
}

func (s *smtpServer) Stop() error {
	return s.server.Close()
}

// backend implements gosmtp.Backend.
type backend struct {
	eventService EventStorer
}

func (b *backend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	return &session{backend: b}, nil
}

// session implements gosmtp.Session.
type session struct {
	backend *backend
	from    string
	to      []string
}

func (s *session) AuthPlain(username, password string) error { return nil }
func (s *session) Mail(from string, opts *gosmtp.MailOptions) error {
	s.from = from
	return nil
}
func (s *session) Rcpt(to string, opts *gosmtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *session) Data(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	parsed, err := parseEmail(raw, s.to)
	if err != nil {
		slog.Error("smtp: failed to parse email", "err", err)
		return nil
	}

	payload, _ := json.Marshal(parsed)

	inc := &event.Incoming{
		Type:    "smtp",
		Payload: json.RawMessage(payload),
	}

	if err := s.backend.eventService.HandleIncoming(context.Background(), inc); err != nil {
		slog.Error("smtp: failed to store event", "err", err)
	}
	return nil
}

func (s *session) Reset()        { s.from = ""; s.to = nil }
func (s *session) Logout() error { return nil }

// EmailAddress represents an email address.
type EmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Attachment represents a parsed email attachment.
type Attachment struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
	Type     string `json:"type"`
}

// ParsedEmail is the structure stored as event payload.
type ParsedEmail struct {
	Subject       string         `json:"subject"`
	From          []EmailAddress `json:"from"`
	To            []EmailAddress `json:"to"`
	Cc            []EmailAddress `json:"cc"`
	ReplyTo       []EmailAddress `json:"reply_to"`
	AllRecipients []string       `json:"all_recipients"`
	TextBody      string         `json:"text_body"`
	HTMLBody      string         `json:"html_body"`
	Raw           string         `json:"raw"`
	Attachments   []Attachment   `json:"attachments"`
}

func parseEmail(raw []byte, recipients []string) (*ParsedEmail, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	parsed := &ParsedEmail{
		Subject:       msg.Header.Get("Subject"),
		Raw:           string(raw),
		From:          parseAddresses(msg.Header, "From"),
		To:            parseAddresses(msg.Header, "To"),
		Cc:            parseAddresses(msg.Header, "Cc"),
		ReplyTo:       parseAddresses(msg.Header, "Reply-To"),
		AllRecipients: recipients,
		Attachments:   []Attachment{},
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		body, _ := io.ReadAll(msg.Body)
		decoded := decodeContent(body, msg.Header.Get("Content-Transfer-Encoding"))
		if strings.HasPrefix(mediaType, "text/html") {
			parsed.HTMLBody = string(decoded)
		} else {
			parsed.TextBody = string(decoded)
		}
		return parsed, nil
	}

	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		processPart(part, parsed)
	}

	return parsed, nil
}

func processPart(part *multipart.Part, parsed *ParsedEmail) {
	disposition := part.Header.Get("Content-Disposition")
	ct := part.Header.Get("Content-Type")

	if strings.HasPrefix(disposition, "attachment") || strings.HasPrefix(disposition, "inline") {
		content, _ := io.ReadAll(part)
		encoding := part.Header.Get("Content-Transfer-Encoding")
		if strings.EqualFold(encoding, "base64") {
			if decoded, err := base64.StdEncoding.DecodeString(string(content)); err == nil {
				content = decoded
			}
		}

		filename := part.FileName()
		if filename == "" {
			filename = "unnamed"
		}
		mimeType := ct
		if idx := strings.Index(mimeType, ";"); idx > 0 {
			mimeType = strings.TrimSpace(mimeType[:idx])
		}
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}

		parsed.Attachments = append(parsed.Attachments, Attachment{
			Filename: filename,
			Content:  base64.StdEncoding.EncodeToString(content),
			Type:     mimeType,
		})
		return
	}

	mediaType, _, _ := mime.ParseMediaType(ct)
	body, _ := io.ReadAll(part)
	decoded := decodeContent(body, part.Header.Get("Content-Transfer-Encoding"))

	if strings.HasPrefix(mediaType, "text/html") {
		parsed.HTMLBody += string(decoded)
	} else {
		parsed.TextBody += string(decoded)
	}
}

func parseAddresses(header mail.Header, key string) []EmailAddress {
	addrs, err := header.AddressList(key)
	if err != nil {
		return []EmailAddress{}
	}
	result := make([]EmailAddress, len(addrs))
	for i, a := range addrs {
		result[i] = EmailAddress{Email: a.Address, Name: a.Name}
	}
	return result
}

func decodeContent(data []byte, encoding string) []byte {
	switch strings.ToLower(encoding) {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return data
		}
		return decoded
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(data)))
		if err != nil {
			return data
		}
		return decoded
	default:
		return data
	}
}
