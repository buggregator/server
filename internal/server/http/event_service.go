package http

import (
	"context"
	"log/slog"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	"github.com/buggregator/go-buggregator/internal/server/ws"
)

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
	ev := event.NewEvent(inc)

	if err := s.store.Store(ctx, ev); err != nil {
		return err
	}

	slog.Info("event stored", "uuid", ev.UUID, "type", ev.Type, "project", ev.Project)

	// Build preview and broadcast.
	preview := s.registry.Previews().BuildPreview(ev)

	channel := "events"
	if ev.Project != "" {
		s.hub.Broadcast("events.project."+ev.Project, "event.received", preview)
	}
	s.hub.Broadcast(channel, "event.received", preview)

	// Notify modules (e.g., webhooks).
	s.registry.NotifyEventStored(ev)

	return nil
}

// BroadcastDeleted broadcasts event deletion.
func (s *EventService) BroadcastDeleted(uuid, project string) {
	data := map[string]string{"uuid": uuid, "project": project}
	if project != "" {
		s.hub.Broadcast("events.project."+project, "event.deleted", data)
	}
	s.hub.Broadcast("events", "event.deleted", data)
}

// BroadcastCleared broadcasts events cleared.
func (s *EventService) BroadcastCleared(eventType, project string) {
	if project != "" {
		s.hub.Broadcast("events.project."+project, "events.cleared", nil)
	}
	s.hub.Broadcast("events", "events.cleared", nil)
}
