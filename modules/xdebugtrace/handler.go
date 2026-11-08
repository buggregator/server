package xdebugtrace

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/buggregator/go-buggregator/internal/event"
)

// HeaderDetectedType is set by the ingestion pipeline from
// X-Buggregator-Event or the DSN userinfo (xdebug-trace@host).
const headerDetectedType = "X-Buggregator-Detected-Type"

type handler struct {
	dir    string
	limits Limits
}

func (h *handler) Priority() int { return 41 }

func (h *handler) Match(r *http.Request) bool {
	return r.Method == http.MethodPost && r.Header.Get(headerDetectedType) == Type
}

// Handle streams the body to a file and summarizes it in the same pass:
// the trace is never held in memory whole.
func (h *handler) Handle(r *http.Request) (*event.Incoming, error) {
	defer r.Body.Close()
	uuid := event.GenerateUUID()
	part := filepath.Join(h.dir, uuid+extPart)
	f, err := os.OpenFile(part, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("xdebug-trace: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(part)
		}
	}()

	body := &countReader{r: io.LimitReader(r.Body, h.limits.MaxUpload+1)}
	sniff := &sniffWriter{w: f}
	tee := io.TeeReader(body, sniff)

	sum, err := Summarize(tee, h.limits.MaxSize)
	if err != nil {
		if errors.Is(err, ErrNotFormat1) {
			return nil, err
		}
		if body.n <= h.limits.MaxUpload {
			return nil, fmt.Errorf("xdebug-trace: read trace: %w", err)
		}
	}
	if _, err := io.Copy(io.Discard, tee); err != nil {
		return nil, fmt.Errorf("xdebug-trace: read upload: %w", err)
	}
	if body.n > h.limits.MaxUpload {
		return nil, fmt.Errorf("xdebug-trace: upload is larger than %d bytes (xdebug_trace.max_upload)", h.limits.MaxUpload)
	}
	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("xdebug-trace: %w", err)
	}

	gz := sniff.gzip()
	name := uuid + extPlain
	if gz {
		name = uuid + extGzip
	}

	index := sum.Index
	sum.Index = nil
	if len(index) > 0 {
		b, _ := json.Marshal(index)
		if err := os.WriteFile(filepath.Join(h.dir, uuid+extIndex), b, 0o644); err != nil {
			return nil, fmt.Errorf("xdebug-trace: write index: %w", err)
		}
	}
	if err := os.Rename(part, filepath.Join(h.dir, name)); err != nil {
		os.Remove(filepath.Join(h.dir, uuid+extIndex))
		return nil, fmt.Errorf("xdebug-trace: %w", err)
	}
	ok = true

	p := Payload{
		File:       name,
		Name:       headerValue(r, "X-Buggregator-Trace-Name"),
		URL:        headerValue(r, "X-Buggregator-Trace-Url"),
		Size:       body.n,
		Compressed: gz,
		Summary:    sum,
	}
	b, _ := json.Marshal(p)
	return &event.Incoming{UUID: uuid, Type: Type, Payload: b}, nil
}

// headerValue trims a client header to a sane length.
func headerValue(r *http.Request, name string) string {
	v := strings.TrimSpace(r.Header.Get(name))
	if len(v) > 2048 {
		v = v[:2048]
	}
	return v
}

type countReader struct {
	r io.Reader
	n int64
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// sniffWriter keeps the first two bytes to tell gzip from plain text.
type sniffWriter struct {
	w    io.Writer
	head []byte
}

func (s *sniffWriter) Write(p []byte) (int, error) {
	if len(s.head) < 2 {
		s.head = append(s.head, p[:min(2-len(s.head), len(p))]...)
	}
	return s.w.Write(p)
}

func (s *sniffWriter) gzip() bool {
	return len(s.head) == 2 && s.head[0] == 0x1f && s.head[1] == 0x8b
}
