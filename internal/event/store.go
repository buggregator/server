package event

import (
	"context"
	"encoding/json"
)

// FindOptions configures event listing queries.
type FindOptions struct {
	Type    string
	Project string
	Limit   int
	Offset  int

	// From and To bound the selection by event time (unix seconds with a
	// fraction, same as Event.Timestamp). Zero means "no bound".
	From float64
	To   float64

	// PinnedFirst puts pinned events before the rest, so a limit never
	// pushes a pinned event out of the list.
	PinnedFirst bool
}

// DeleteOptions configures batch deletion.
type DeleteOptions struct {
	Type    string
	Project string
	UUIDs   []string
	// KeepPinned leaves pinned events in place.
	KeepPinned bool
}

// Store persists and retrieves events.
type Store interface {
	Store(ctx context.Context, ev Event) error
	FindByUUID(ctx context.Context, uuid string) (*Event, error)
	FindAll(ctx context.Context, opts FindOptions) ([]Event, error)
	Delete(ctx context.Context, uuid string) error
	DeleteAll(ctx context.Context, opts DeleteOptions) error
	// UpdatePayload replaces the payload of a stored event; ErrNotFound when
	// there is none.
	UpdatePayload(ctx context.Context, uuid string, payload json.RawMessage) error
	Pin(ctx context.Context, uuid string) error
	Unpin(ctx context.Context, uuid string) error
}
