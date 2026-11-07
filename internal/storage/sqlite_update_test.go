package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
)

func TestSQLiteStore_UpdatePayloadMissing(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := NewMigrator(db)
	if err := m.AddFromFS("core", CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	s := NewSQLiteStore(db)
	if err := s.UpdatePayload(context.Background(), "nope", []byte(`{}`)); !errors.Is(err, event.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	ev := event.Event{UUID: "e", Type: "ray", Payload: []byte(`{"a":1}`), Project: "default"}
	if err := s.Store(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdatePayload(context.Background(), "e", []byte(`{"a":2}`)); err != nil {
		t.Errorf("update: %v", err)
	}
}
