package http

import (
	"context"
	"log/slog"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/ws"
)

const defaultProject = "default"

// EventService handles the store → preview → broadcast → notify pipeline.
type EventService struct {
	store    event.Store
	hub      *ws.Hub
	registry *module.Registry
}

func NewEventService(store event.Store, hub *ws.Hub, registry *module.Registry) *EventService {
	return &EventService{store: store, hub: hub, registry: registry}
}

// HandleIncoming stores an event, broadcasts preview, and notifies modules.
func (s *EventService) HandleIncoming(ctx context.Context, inc *event.Incoming) error {
	// Assign default project if not set.
	if inc.Project == "" {
		inc.Project = defaultProject
	}

	ev := event.NewEvent(inc)

	if err := s.store.Store(ctx, ev); err != nil {
		return err
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
