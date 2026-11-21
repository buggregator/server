// Package broadcast captures events an app sends to a Pusher-compatible
// server (Laravel Reverb, Pusher, soketi) over the Pusher HTTP API:
// POST /apps/{app_id}/events and /batch_events. Point the app's REVERB_HOST /
// PUSHER_HOST at Buggregator; each event becomes a card with its channels and
// data. With an upstream set, the same request also goes to the real server,
// so browsers still get the events.
package broadcast

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
)

// Config is the module's settings.
type Config struct {
	// Upstream is the real Pusher-compatible server (e.g. http://127.0.0.1:8080
	// for Reverb). Empty: capture only, the app gets {} back.
	Upstream string
	// Secrets maps an app key to its secret, to check request signatures.
	// A key with no secret is not checked.
	Secrets map[string]string
	// Timeout bounds a forward; default 5 s.
	Timeout time.Duration
}

// EventStorer stores one event: a batch request makes several.
type EventStorer interface {
	HandleIncoming(ctx context.Context, inc *event.Incoming) error
}

// Module is the broadcast events catcher.
type Module struct {
	module.BaseModule
	h *handler
}

// New returns the module.
func New(cfg Config) *Module {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	cfg.Upstream = strings.TrimRight(strings.TrimSpace(cfg.Upstream), "/")
	return &Module{h: &handler{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}}
}

func (m *Module) Name() string { return "Broadcast" }
func (m *Module) Type() string { return "broadcast" }

func (m *Module) HTTPHandler() module.HTTPIngestionHandler { return m.h }

func (m *Module) PreviewMapper() event.PreviewMapper { return &previewMapper{} }

// SetEventService sets where the events go.
func (m *Module) SetEventService(es EventStorer) { m.h.es = es }
