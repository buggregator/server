package http

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/metrics"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/ws"
)

const defaultProject = "default"

// EventService handles the store -> preview -> broadcast -> notify pipeline.
type EventService struct {
	store    event.Store
	hub      *ws.Hub
	registry *module.Registry
	metrics  *metrics.Collector
	db       *sql.DB // optional; used to auto-register new project keys
}

func NewEventService(store event.Store, hub *ws.Hub, registry *module.Registry, m *metrics.Collector, db *sql.DB) *EventService {
	return &EventService{store: store, hub: hub, registry: registry, metrics: m, db: db}
}

// HandleIncoming stores an event, broadcasts preview, and notifies modules.
func (s *EventService) HandleIncoming(ctx context.Context, inc *event.Incoming) error {
	// Assign default project if not set.
	if inc.Project == "" {
		inc.Project = defaultProject
	}

	// Auto-register the project so the frontend can see and switch to it.
	// Sentry events arrive with a project key derived from the DSN path
	// (e.g. /api/123/envelope/ → "123"), which won't exist in the projects
	// table seeded only with "default". Without this row, the frontend filters
	// the event out of the events list and the sidebar dot stays unlit.
	if s.db != nil && inc.Project != defaultProject {
		if _, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO projects (key, name) VALUES (?, ?)`,
			inc.Project, inc.Project,
		); err != nil {
			slog.Warn("failed to register project", "key", inc.Project, "err", err)
		}
	}

	ev := event.NewEvent(inc)

	start := time.Now()
	if err := s.store.Store(ctx, ev); err != nil {
		if s.metrics != nil {
			s.metrics.EventsIngestionErrors.WithLabelValues(ev.Type, "store_failed").Inc()
		}
		return err
	}
	duration := time.Since(start).Seconds()

	if s.metrics != nil {
		s.metrics.EventsReceivedTotal.WithLabelValues(ev.Type, ev.Project).Inc()
		s.metrics.EventsIngestionDuration.WithLabelValues(ev.Type).Observe(duration)
		s.metrics.EventsPayloadBytes.WithLabelValues(ev.Type).Observe(float64(len(ev.Payload)))
	}

	slog.Info("event stored", "uuid", ev.UUID, "type", ev.Type, "project", ev.Project)

	// Build preview and broadcast to project-specific channel.
	preview := s.registry.Previews().BuildPreview(ev)
	s.hub.Broadcast("events.project."+ev.Project, "event.received", preview)

	// Notify modules (e.g., webhooks).
	s.registry.NotifyEventStored(ev)

	return nil
}

// BroadcastDeleted broadcasts event deletion.
func (s *EventService) BroadcastDeleted(uuid, project string) {
	if project == "" {
		project = defaultProject
	}
	data := map[string]string{"uuid": uuid, "project": project}
	s.hub.Broadcast("events.project."+project, "event.deleted", data)
}

// BroadcastCleared broadcasts events cleared.
func (s *EventService) BroadcastCleared(eventType, project string) {
	if project == "" {
		project = defaultProject
	}
	s.hub.Broadcast("events.project."+project, "events.cleared", nil)
}
