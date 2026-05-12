package smtp

import (
	"context"
	"database/sql"
	"net/http"
	"sync"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/tcp"
	"github.com/buggregator/go-buggregator/internal/storage"
)

// waiter holds a long-poll subscription waiting for a matching SMTP event.
type waiter struct {
	filter MessageFilter
	ch     chan event.Event
}

type Module struct {
	module.BaseModule
	addr         string
	eventService EventStorer
	attachments  *storage.AttachmentStore
	db           *sql.DB

	mu      sync.Mutex
	waiters []*waiter
}

type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

func New(addr string, attachments *storage.AttachmentStore, db *sql.DB) *Module {
	return &Module{addr: addr, attachments: attachments, db: db}
}

func (m *Module) SetEventService(es EventStorer) {
	m.eventService = es
}

func (m *Module) Name() string { return "SMTP" }
func (m *Module) Type() string { return "smtp" }

func (m *Module) TCPServers() []tcp.ServerConfig {
	if m.eventService == nil {
		return nil
	}
	return []tcp.ServerConfig{
		{
			Name:    "smtp",
			Address: m.addr,
			Starter: newSMTPServer(m.addr, m.eventService, m.attachments, m.db),
		},
	}
}

// RegisterRoutes registers the SMTP testing API endpoints.
func (m *Module) RegisterRoutes(mux *http.ServeMux, store event.Store) {
	registerSMTPAPI(mux, store, m)
}

// OnEventStored notifies any long-poll waiters when a new SMTP event arrives.
func (m *Module) OnEventStored(ev event.Event) {
	if ev.Type != "smtp" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, w := range m.waiters {
		if matchesFilter(ev, w.filter) {
			select {
			case w.ch <- ev:
			default:
				// Channel already has an event; waiter will pick it up.
			}
		}
	}
}

// subscribe registers a waiter for the wait endpoint and returns a channel and
// an unsubscribe function. The caller must call unsubscribe when done.
func (m *Module) subscribe(f MessageFilter) (<-chan event.Event, func()) {
	ch := make(chan event.Event, 1)
	w := &waiter{filter: f, ch: ch}
	m.mu.Lock()
	m.waiters = append(m.waiters, w)
	m.mu.Unlock()
	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		for i, ww := range m.waiters {
			if ww == w {
				m.waiters = append(m.waiters[:i], m.waiters[i+1:]...)
				return
			}
		}
	}
}

func (m *Module) PreviewMapper() event.PreviewMapper {
	return &previewMapper{}
}
