package httpdumps

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/storage"
)

type handler struct {
	attachments *storage.AttachmentStore
	db          *sql.DB
}

func (h *handler) Priority() int { return 9999 }

func (h *handler) Match(r *http.Request) bool {
	return true
}

func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	eventUUID := event.GenerateUUID()

	// Headers as arrays (PSR-7 format).
	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[k] = v
	}

	// Parse query params.
	query := make(map[string]any)
	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			query[k] = v[0]
		} else {
			query[k] = v
		}
	}

	cookies := make(map[string]string)
	for _, c := range r.Cookies() {
		cookies[c.Name] = c.Value
	}

	uri := strings.TrimPrefix(r.URL.Path, "/")
	ct := r.Header.Get("Content-Type")
	post := make(map[string]any)
	var bodyStr string
	var files []map[string]any

	if strings.Contains(ct, "multipart/form-data") {
		// Parse multipart — extract files and form fields.
		r.ParseMultipartForm(10 << 20) // 10MB max
		if r.MultipartForm != nil {
			// Form values → post.
			for k, v := range r.MultipartForm.Value {
				if len(v) == 1 {
					post[k] = v[0]
				} else {
					post[k] = v
				}
			}

			// Files → store and collect metadata.
			for _, fileHeaders := range r.MultipartForm.File {
				for _, fh := range fileHeaders {
					f, err := fh.Open()
					if err != nil {
						continue
					}
					content, err := io.ReadAll(f)
					f.Close()
					if err != nil {
						continue
					}

					attUUID := event.GenerateUUID()
					path := eventUUID + "/" + fh.Filename
					mime := fh.Header.Get("Content-Type")
					if mime == "" {
						mime = "application/octet-stream"
					}

					if h.attachments != nil {
						h.attachments.Store(path, content)
					}
					if h.db != nil {
						h.db.Exec(
							`INSERT INTO http_dump_attachments (uuid, event_uuid, name, path, size, mime) VALUES (?, ?, ?, ?, ?, ?)`,
							attUUID, eventUUID, fh.Filename, path, len(content), mime,
						)
					}

					files = append(files, map[string]any{
						"uuid": attUUID,
						"name": fh.Filename,
						"size": len(content),
						"mime": mime,
						"uri":  "/api/http-dump/" + eventUUID + "/attachments/" + attUUID,
					})
				}
			}
		}
	} else {
		// Non-multipart — read body as string.
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		bodyStr = string(body)

		if strings.Contains(ct, "application/x-www-form-urlencoded") {
			r.ParseForm()
			for k, v := range r.PostForm {
				if len(v) == 1 {
					post[k] = v[0]
				} else {
					post[k] = v
				}
			}
		} else if strings.Contains(ct, "application/json") && len(body) > 0 {
			var parsed map[string]any
			if json.Unmarshal(body, &parsed) == nil {
				post = parsed
			}
		}
	}

	if files == nil {
		files = []map[string]any{}
	}

	payload := map[string]any{
		"received_at": time.Now().Format("2006-01-02 15:04:05"),
		"host":        r.Host,
		"request": map[string]any{
			"method":  r.Method,
			"uri":     uri,
			"headers": headers,
			"body":    bodyStr,
			"query":   query,
			"post":    post,
			"cookies": cookies,
			"files":   files,
		},
	}

	b, _ := json.Marshal(payload)

	return &event.Incoming{
		UUID:    eventUUID,
		Type:    "http-dump",
		Payload: json.RawMessage(b),
	}, nil
}
