package event

import "context"

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
}

// DeleteOptions configures batch deletion.
type DeleteOptions struct {
	Type    string
	Project string
	UUIDs   []string
}

// Store persists and retrieves events.
type Store interface {
	Store(ctx context.Context, ev Event) error
	FindByUUID(ctx context.Context, uuid string) (*Event, error)
	FindAll(ctx context.Context, opts FindOptions) ([]Event, error)
	Delete(ctx context.Context, uuid string) error
	DeleteAll(ctx context.Context, opts DeleteOptions) error
	Pin(ctx context.Context, uuid string) error
	Unpin(ctx context.Context, uuid string) error
}
