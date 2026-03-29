package storage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/storage"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func makeEvent(uuid, typ, project string) event.Event {
	return event.Event{
		UUID:      uuid,
		Type:      typ,
		Payload:   json.RawMessage(`{"test":true}`),
		Timestamp: 1700000000.123456,
		Project:   project,
	}
}

func TestSQLiteStore_Store_and_FindByUUID(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	ev := makeEvent("uuid-1", "sentry", "default")
	if err := store.Store(ctx, ev); err != nil {
		t.Fatalf("Store: %v", err)
	}

	found, err := store.FindByUUID(ctx, "uuid-1")
	if err != nil {
		t.Fatalf("FindByUUID: %v", err)
	}
	if found == nil {
		t.Fatal("expected event, got nil")
	}
	if found.UUID != "uuid-1" {
		t.Errorf("UUID = %q, want %q", found.UUID, "uuid-1")
	}
	if found.Type != "sentry" {
		t.Errorf("Type = %q, want %q", found.Type, "sentry")
	}
	if found.Project != "default" {
		t.Errorf("Project = %q, want %q", found.Project, "default")
	}
}

func TestSQLiteStore_FindByUUID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)

	found, err := store.FindByUUID(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("FindByUUID: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %+v", found)
	}
}

func TestSQLiteStore_FindAll(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	events := []event.Event{
		{UUID: "e1", Type: "sentry", Payload: json.RawMessage(`{}`), Timestamp: 1700000001.0, Project: "proj-a"},
		{UUID: "e2", Type: "ray", Payload: json.RawMessage(`{}`), Timestamp: 1700000002.0, Project: "proj-a"},
		{UUID: "e3", Type: "sentry", Payload: json.RawMessage(`{}`), Timestamp: 1700000003.0, Project: "proj-b"},
	}
	for _, ev := range events {
		if err := store.Store(ctx, ev); err != nil {
			t.Fatalf("Store: %v", err)
		}
	}

	t.Run("all events", func(t *testing.T) {
		all, err := store.FindAll(ctx, event.FindOptions{})
		if err != nil {
			t.Fatalf("FindAll: %v", err)
		}
		if len(all) != 3 {
			t.Errorf("len = %d, want 3", len(all))
		}
		// Should be ordered by timestamp DESC
		if all[0].UUID != "e3" {
			t.Errorf("first event UUID = %q, want %q", all[0].UUID, "e3")
		}
	})

	t.Run("filter by type", func(t *testing.T) {
		result, err := store.FindAll(ctx, event.FindOptions{Type: "sentry"})
		if err != nil {
			t.Fatalf("FindAll: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
	})

	t.Run("filter by project", func(t *testing.T) {
		result, err := store.FindAll(ctx, event.FindOptions{Project: "proj-a"})
		if err != nil {
			t.Fatalf("FindAll: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("len = %d, want 2", len(result))
		}
	})

	t.Run("filter by type and project", func(t *testing.T) {
		result, err := store.FindAll(ctx, event.FindOptions{Type: "sentry", Project: "proj-a"})
		if err != nil {
			t.Fatalf("FindAll: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len = %d, want 1", len(result))
		}
	})

	t.Run("limit and offset", func(t *testing.T) {
		result, err := store.FindAll(ctx, event.FindOptions{Limit: 1})
		if err != nil {
			t.Fatalf("FindAll: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len = %d, want 1", len(result))
		}

		result, err = store.FindAll(ctx, event.FindOptions{Limit: 1, Offset: 1})
		if err != nil {
			t.Fatalf("FindAll: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("len = %d, want 1", len(result))
		}
		if result[0].UUID != "e2" {
			t.Errorf("UUID = %q, want %q", result[0].UUID, "e2")
		}
	})
}

func TestSQLiteStore_Delete(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	ev := makeEvent("del-1", "sentry", "default")
	store.Store(ctx, ev)

	if err := store.Delete(ctx, "del-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	found, _ := store.FindByUUID(ctx, "del-1")
	if found != nil {
		t.Error("expected event to be deleted")
	}
}

func TestSQLiteStore_DeleteAll(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	for _, ev := range []event.Event{
		makeEvent("d1", "sentry", "proj-a"),
		makeEvent("d2", "ray", "proj-a"),
		makeEvent("d3", "sentry", "proj-b"),
	} {
		store.Store(ctx, ev)
	}

	t.Run("delete by UUIDs", func(t *testing.T) {
		if err := store.DeleteAll(ctx, event.DeleteOptions{UUIDs: []string{"d1"}}); err != nil {
			t.Fatal(err)
		}
		all, _ := store.FindAll(ctx, event.FindOptions{})
		if len(all) != 2 {
			t.Errorf("len = %d, want 2", len(all))
		}
	})

	t.Run("delete by type", func(t *testing.T) {
		if err := store.DeleteAll(ctx, event.DeleteOptions{Type: "sentry"}); err != nil {
			t.Fatal(err)
		}
		all, _ := store.FindAll(ctx, event.FindOptions{})
		if len(all) != 1 {
			t.Errorf("len = %d, want 1", len(all))
		}
	})

	t.Run("delete all", func(t *testing.T) {
		if err := store.DeleteAll(ctx, event.DeleteOptions{}); err != nil {
			t.Fatal(err)
		}
		all, _ := store.FindAll(ctx, event.FindOptions{})
		if len(all) != 0 {
			t.Errorf("len = %d, want 0", len(all))
		}
	})
}

func TestSQLiteStore_Pin_Unpin(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	store.Store(ctx, makeEvent("pin-1", "sentry", "default"))

	if err := store.Pin(ctx, "pin-1"); err != nil {
		t.Fatalf("Pin: %v", err)
	}
	found, _ := store.FindByUUID(ctx, "pin-1")
	if !found.IsPinned {
		t.Error("expected IsPinned to be true")
	}

	if err := store.Unpin(ctx, "pin-1"); err != nil {
		t.Fatalf("Unpin: %v", err)
	}
	found, _ = store.FindByUUID(ctx, "pin-1")
	if found.IsPinned {
		t.Error("expected IsPinned to be false")
	}
}
