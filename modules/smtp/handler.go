package smtp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"strings"
	"time"

	"database/sql"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/storage"
	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"
	"golang.org/x/text/encoding/ianaindex"
)

// smtpServer wraps go-smtp and implements tcp.Starter.
type smtpServer struct {
	server       *gosmtp.Server
	eventService EventStorer
	attachments  *storage.AttachmentStore
	db           *sql.DB
}

func newSMTPServer(addr string, es EventStorer, att *storage.AttachmentStore, db *sql.DB) *smtpServer {
	be := &backend{eventService: es, attachments: att, db: db}
	s := gosmtp.NewServer(be)
	s.Addr = addr
	s.Domain = "localhost"
	s.ReadTimeout = 60 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 10 * 1024 * 1024
	s.AllowInsecureAuth = true

	return &smtpServer{server: s, eventService: es, attachments: att, db: db}
}

// Start binds the port before it returns, so a busy or bad address fails
// startup like the other TCP servers; serving runs in the background.
func (s *smtpServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("smtp: listen on %s: %w", s.server.Addr, err)
	}
	go func() {
		if err := s.server.Serve(ln); err != nil && err != gosmtp.ErrServerClosed {
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
	attachments  *storage.AttachmentStore
	db           *sql.DB
}

func (b *backend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	return &session{backend: b}, nil
}

// session implements gosmtp.Session.
type session struct {
	backend *backend
	from    string
	to      []string
	project string // extracted from SMTP AUTH username
}

// AuthMechanisms implements gosmtp.AuthSession. go-smtp v0.24 only offers
// AUTH through this interface; the username selects the project (#343).
func (s *session) AuthMechanisms() []string {
	return []string{sasl.Plain, sasl.Login}
}

// Auth implements gosmtp.AuthSession. Any credentials are accepted.
func (s *session) Auth(mech string) (sasl.Server, error) {
	switch mech {
	case sasl.Plain:
		return sasl.NewPlainServer(func(identity, username, password string) error {
			s.setProject(username)
			return nil
		}), nil
	case sasl.Login:
		return &loginServer{auth: func(username, password string) error {
			s.setProject(username)
			return nil
		}}, nil
	}
	return nil, gosmtp.ErrAuthUnknownMechanism
}

// setProject takes the project key from an AUTH username:
// "project@host" or plain "project".
func (s *session) setProject(username string) {
	if idx := strings.Index(username, "@"); idx > 0 {
		username = username[:idx]
	}
	s.project = username
}

// loginServer is a server side of the LOGIN mechanism, which go-sasl only
// implements for clients.
type loginServer struct {
	auth     func(username, password string) error
	username string
	step     int
}

func (l *loginServer) Next(response []byte) (challenge []byte, done bool, err error) {
	switch l.step {
	case 0:
		l.step++
		if len(response) == 0 { // no initial response
			return []byte("Username:"), false, nil
		}
		l.username = string(response)
		l.step++
		return []byte("Password:"), false, nil
	case 1:
		l.username = string(response)
		l.step++
		return []byte("Password:"), false, nil
	case 2:
		l.step++
		return nil, true, l.auth(l.username, string(response))
	}
	return nil, true, sasl.ErrUnexpectedClientResponse
}

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

	parsed, attachments, err := parseEmail(raw, s.to)
	if err != nil {
		slog.Error("smtp: failed to parse email", "err", err)
		return nil
	}

	// Generate event UUID for attachment storage.
	eventUUID := event.GenerateUUID()

	// Store attachments and collect CID→URL mappings for inline images.
	cidMap := make(map[string]string) // cid -> preview URL
	if s.backend.attachments != nil && len(attachments) > 0 {
		for _, att := range attachments {
			path := eventUUID + "/" + att.Filename
			if err := s.backend.attachments.Store(path, att.content); err != nil {
				slog.Error("smtp: failed to store attachment", "err", err)
				continue
			}
			attUUID := event.GenerateUUID()
			if s.backend.db != nil {
				s.backend.db.Exec(
					`INSERT INTO smtp_attachments (uuid, event_uuid, name, path, size, mime, content_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					attUUID, eventUUID, att.Filename, path, len(att.content), att.Type, att.ContentID,
				)
			}
			if att.ContentID != "" {
				cid := strings.Trim(att.ContentID, "<>")
				cidMap[cid] = "/api/smtp/attachments/" + eventUUID + "/preview/" + attUUID
			}
		}
	}

	// Replace cid: references in HTML with attachment preview URLs.
	if parsed.HTML != "" && len(cidMap) > 0 {
		for cid, url := range cidMap {
			parsed.HTML = strings.ReplaceAll(parsed.HTML, "cid:"+cid, url)
		}
	}

	payload, _ := json.Marshal(parsed)

	inc := &event.Incoming{
		UUID:    eventUUID,
		Type:    "smtp",
		Payload: json.RawMessage(payload),
		Project: s.project,
	}

	if err := s.backend.eventService.HandleIncoming(context.Background(), inc); err != nil {
		slog.Error("smtp: failed to store event", "err", err)
	}
	return nil
}

// Reset runs after every message; AUTH is per connection, so the project stays.
func (s *session) Reset()        { s.from = ""; s.to = nil }
func (s *session) Logout() error { return nil }

// EmailAddress represents an email address.
type EmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// parsedAttachment holds raw attachment data during parsing.
type parsedAttachment struct {
	Filename  string
	Type      string
	ContentID string
	content   []byte // raw binary content
}

// ParsedEmail is the structure stored as event payload.
// Field names match the original PHP Buggregator Message::jsonSerialize().
type ParsedEmail struct {
	ID      *string        `json:"id"`
	Subject string         `json:"subject"`
	From    []EmailAddress `json:"from"`
	To      []EmailAddress `json:"to"`
	Cc      []EmailAddress `json:"cc"`
	Bcc     []string       `json:"bcc"`
	ReplyTo []EmailAddress `json:"reply_to"`
	Text    string         `json:"text"`
	HTML    string         `json:"html"`
	Raw     string         `json:"raw"`
}

func parseEmail(raw []byte, recipients []string) (*ParsedEmail, []parsedAttachment, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, err
	}

	// Parse Message-ID.
	var msgID *string
	if id := msg.Header.Get("Message-ID"); id != "" {
		msgID = &id
	}

	to := parseAddresses(msg.Header, "To")
	cc := parseAddresses(msg.Header, "Cc")

	// Calculate BCC: recipients not in To or CC.
	visibleEmails := make(map[string]bool)
	for _, addr := range to {
		visibleEmails[addr.Email] = true
	}
	for _, addr := range cc {
		visibleEmails[addr.Email] = true
	}
	var bcc []string
	for _, r := range recipients {
		if !visibleEmails[r] {
			bcc = append(bcc, r)
		}
	}
	if bcc == nil {
		bcc = []string{}
	}

	// Decode RFC 2047 encoded-word in Subject header (e.g. =?utf-8?Q?...?=).
	subject := msg.Header.Get("Subject")
	dec := new(mime.WordDecoder)
	if decoded, err := dec.DecodeHeader(subject); err == nil {
		subject = decoded
	}

	parsed := &ParsedEmail{
		ID:      msgID,
		Subject: subject,
		Raw:     string(raw),
		From:    parseAddresses(msg.Header, "From"),
		To:      to,
		Cc:      cc,
		Bcc:     bcc,
		ReplyTo: parseAddresses(msg.Header, "Reply-To"),
	}

	contentType := msg.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		body, _ := io.ReadAll(msg.Body)
		decoded := decodeContent(body, msg.Header.Get("Content-Transfer-Encoding"))
		decoded = convertToUTF8(decoded, params["charset"])
		if strings.HasPrefix(mediaType, "text/html") {
			parsed.HTML = string(decoded)
		} else {
			parsed.Text = string(decoded)
		}
		return parsed, nil, nil
	}

	var atts []parsedAttachment
	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		processPart(part, parsed, &atts)
	}

	return parsed, atts, nil
}

func processPart(part *multipart.Part, parsed *ParsedEmail, atts *[]parsedAttachment) {
	ct := part.Header.Get("Content-Type")
	disposition := part.Header.Get("Content-Disposition")

	mediaType, params, _ := mime.ParseMediaType(ct)

	// Recurse into nested multipart.
	if strings.HasPrefix(mediaType, "multipart/") {
		if boundary := params["boundary"]; boundary != "" {
			mr := multipart.NewReader(part, boundary)
			for {
				subpart, err := mr.NextPart()
				if err != nil {
					break
				}
				processPart(subpart, parsed, atts)
			}
		}
		return
	}

	// Attachment.
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

		contentID := strings.Trim(part.Header.Get("Content-ID"), "<>")

		*atts = append(*atts, parsedAttachment{
			Filename:  filename,
			Type:      mimeType,
			ContentID: contentID,
			content:   content,
		})
		return
	}

	// Body content.
	body, _ := io.ReadAll(part)
	decoded := decodeContent(body, part.Header.Get("Content-Transfer-Encoding"))
	decoded = convertToUTF8(decoded, params["charset"])

	if strings.HasPrefix(mediaType, "text/html") {
		parsed.HTML += string(decoded)
	} else if strings.HasPrefix(mediaType, "text/plain") || ct == "" {
		parsed.Text += string(decoded)
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

// convertToUTF8 converts data from the given charset to UTF-8.
// If charset is empty, "utf-8", or "us-ascii", the data is returned as-is.
func convertToUTF8(data []byte, charset string) []byte {
	charset = strings.ToLower(strings.TrimSpace(charset))
	if charset == "" || charset == "utf-8" || charset == "us-ascii" {
		return data
	}
	enc, err := ianaindex.MIME.Encoding(charset)
	if err != nil || enc == nil {
		return data
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return data
	}
	return decoded
}
