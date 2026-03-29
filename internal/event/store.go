package event

import "context"

// FindOptions configures event listing queries.
type FindOptions struct {
	Type    string
	Project string
	Limit   int
	Offset  int
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
