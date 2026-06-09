package http_test

import (
	"context"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
)

func TestEventService_HandleIncoming_AutoRegistersProject(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// Run core migrations so the projects table exists with the default seed row.
	migrator := storage.NewMigrator(db)
	if err := migrator.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Run(); err != nil {
		t.Fatal(err)
	}

	store := storage.NewSQLiteStore(db)
	hub := ws.NewHub()
	registry := module.NewRegistry()
	es := serverhttp.NewEventService(store, hub, registry, nil, db)

	// An event arriving with an explicit non-default project key (e.g. a named
	// DSN path or an X-Buggregator-Project override) must auto-register that key.
	inc := &event.Incoming{
		UUID:    "evt-1",
		Type:    "sentry",
		Payload: []byte(`{"event_id":"evt-1","message":"boom"}`),
		Project: "123",
	}
	if err := es.HandleIncoming(context.Background(), inc); err != nil {
		t.Fatalf("HandleIncoming: %v", err)
	}

	// Project row must exist so the frontend can list/switch to it.
	var name string
	row := db.QueryRow(`SELECT name FROM projects WHERE key = ?`, "123")
	if err := row.Scan(&name); err != nil {
		t.Fatalf("project 123 was not registered: %v", err)
	}

	// Sending another event under the same key must not error (INSERT OR IGNORE).
	inc2 := &event.Incoming{
		UUID:    "evt-2",
		Type:    "sentry",
		Payload: []byte(`{"event_id":"evt-2"}`),
		Project: "123",
	}
	if err := es.HandleIncoming(context.Background(), inc2); err != nil {
		t.Fatalf("HandleIncoming (second event): %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects WHERE key = ?`, "123").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("project 123 row count = %d, want 1", count)
	}
}

func TestEventService_HandleIncoming_DefaultProjectIsNotRewritten(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	migrator := storage.NewMigrator(db)
	if err := migrator.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Run(); err != nil {
		t.Fatal(err)
	}

	// The seeded "default" project row has name "Default" — make sure the
	// auto-register path doesn't overwrite it for vanilla events.
	store := storage.NewSQLiteStore(db)
	es := serverhttp.NewEventService(store, ws.NewHub(), module.NewRegistry(), nil, db)

	inc := &event.Incoming{UUID: "evt-1", Type: "var-dump", Payload: []byte(`{}`)}
	if err := es.HandleIncoming(context.Background(), inc); err != nil {
		t.Fatal(err)
	}

	var name string
	if err := db.QueryRow(`SELECT name FROM projects WHERE key = ?`, "default").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Default" {
		t.Errorf("default project name = %q, want %q", name, "Default")
	}
}
