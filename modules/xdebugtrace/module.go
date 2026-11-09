package xdebugtrace

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
)

// Type is the event type of an uploaded trace.
const Type = "xdebug-trace"

// Defaults for Limits.
const (
	DefaultMaxUpload = 100 << 20
	DefaultMaxSize   = 256 << 20
	DefaultKeep      = 20
)

// Limits bound what an upload may take.
type Limits struct {
	MaxUpload int64 // bytes of the request body (the file as sent, gzip or not)
	MaxSize   int64 // uncompressed bytes read; a longer trace is cut there
	Keep      int   // traces kept; storing one more deletes the oldest
}

// Module stores Xdebug function traces (xdebug.trace_format=1) sent over
// HTTP. The file stays on disk in dir as sent; SQLite keeps the event with
// its summary, and <uuid>.idx.json keeps the call index.
type Module struct {
	module.BaseModule
	dir    string
	limits Limits
	store  event.Store
	db     *sql.DB
}

// New creates the module; dir is where trace files live.
func New(dir string, limits Limits) *Module {
	if limits.MaxUpload <= 0 {
		limits.MaxUpload = DefaultMaxUpload
	}
	if limits.MaxSize <= 0 {
		limits.MaxSize = DefaultMaxSize
	}
	if limits.Keep <= 0 {
		limits.Keep = DefaultKeep
	}
	return &Module{dir: dir, limits: limits}
}

func (m *Module) Name() string { return "Xdebug Trace" }
func (m *Module) Type() string { return Type }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler {
	return &handler{dir: m.dir, limits: m.limits}
}

func (m *Module) PreviewMapper() event.PreviewMapper { return previewMapper{} }

// RegisterRoutes keeps the store for the API and the keep limit.
func (m *Module) RegisterRoutes(_ *http.ServeMux, store event.Store) { m.store = store }

func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, protect func(http.Handler) http.Handler) {
	registerAPI(mux, m, protect)
}

// OnInit creates dir and removes files no event refers to (an upload cut
// by a crash, events deleted while the module was off).
func (m *Module) OnInit(db *sql.DB) error {
	m.db = db
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		// Optional module: uploads fail with this error, the server runs on.
		slog.Error("xdebug-trace: cannot create trace dir", "dir", m.dir, "err", err)
		return nil
	}
	m.sweep()
	return nil
}

func (m *Module) sweep() {
	if m.db == nil {
		return
	}
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		slog.Warn("xdebug-trace: cannot read trace dir", "dir", m.dir, "err", err)
		return
	}
	known := map[string]bool{}
	rows, err := m.db.Query(`SELECT uuid FROM events WHERE type = ?`, Type)
	if err != nil {
		slog.Warn("xdebug-trace: cannot list trace events", "err", err)
		return
	}
	for rows.Next() {
		var u string
		if rows.Scan(&u) == nil {
			known[u] = true
		}
	}
	rows.Close()
	hasFile := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		u, ext, _ := strings.Cut(name, ".")
		if e.IsDir() {
			continue
		}
		if known[u] {
			if "."+ext == extPlain || "."+ext == extGzip {
				hasFile[u] = true
			}
			continue
		}
		if err := os.Remove(filepath.Join(m.dir, name)); err == nil {
			slog.Info("xdebug-trace: removed orphan file", "file", name)
		}
	}
	// Events whose file is gone (memory storage with a file database after
	// a restart) cannot be opened: drop them, and an index left behind.
	for u := range known {
		if hasFile[u] {
			continue
		}
		if _, err := m.db.Exec(`DELETE FROM events WHERE uuid = ?`, u); err == nil {
			os.Remove(filepath.Join(m.dir, u+extIndex))
			slog.Info("xdebug-trace: removed trace event without its file", "uuid", u)
		}
	}
}

// OnEventStored deletes the oldest traces past the keep limit; their files
// go with them through OnEventsDeleted.
func (m *Module) OnEventStored(ev event.Event) {
	if ev.Type != Type || m.store == nil {
		return
	}
	ctx := context.Background()
	old, err := m.store.FindAll(ctx, event.FindOptions{Type: Type, Offset: m.limits.Keep, Limit: 1000})
	if err != nil || len(old) == 0 {
		return
	}
	uuids := make([]string, 0, len(old))
	for _, o := range old {
		if !o.IsPinned {
			uuids = append(uuids, o.UUID)
		}
	}
	if len(uuids) == 0 {
		return
	}
	if err := m.store.DeleteAll(ctx, event.DeleteOptions{UUIDs: uuids, KeepPinned: true}); err != nil {
		slog.Warn("xdebug-trace: failed to delete old traces", "err", err)
	}
}

// OnEventsDeleted removes the files of deleted traces.
func (m *Module) OnEventsDeleted(uuids []string) {
	for _, u := range uuids {
		if !validUUID(u) {
			continue
		}
		for _, ext := range []string{extPlain, extGzip, extIndex} {
			err := os.Remove(filepath.Join(m.dir, u+ext))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				slog.Warn("xdebug-trace: failed to remove file", "uuid", u, "err", err)
			}
		}
	}
}

const (
	extPlain = ".xt"
	extGzip  = ".xt.gz"
	extIndex = ".idx.json"
	extPart  = ".part"
)

// validUUID accepts what event.GenerateUUID makes and nothing that could
// leave dir.
func validUUID(u string) bool {
	if u == "" || len(u) > 64 {
		return false
	}
	for _, c := range u {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' || c == '-') {
			return false
		}
	}
	return true
}

// Payload is the event payload of a trace.
type Payload struct {
	File       string   `json:"file"`       // file name in the trace dir
	Name       string   `json:"name"`       // name the client gave (X-Buggregator-Trace-Name), may be empty
	URL        string   `json:"url"`        // request the trace was recorded for (X-Buggregator-Trace-Url), may be empty
	Size       int64    `json:"size"`       // bytes stored
	Compressed bool     `json:"compressed"` // stored as gzip
	Summary    *Summary `json:"summary"`    // without the index
}

func (m *Module) payload(ctx context.Context, uuid string) (*Payload, error) {
	if !validUUID(uuid) || m.store == nil {
		return nil, os.ErrNotExist
	}
	ev, err := m.store.FindByUUID(ctx, uuid)
	if err != nil || ev == nil || ev.Type != Type {
		return nil, os.ErrNotExist
	}
	var p Payload
	if err := json.Unmarshal(ev.Payload, &p); err != nil {
		return nil, err
	}
	if p.File != uuid+extPlain && p.File != uuid+extGzip {
		return nil, os.ErrNotExist
	}
	return &p, nil
}

var _ module.EventDeleteListener = (*Module)(nil)
