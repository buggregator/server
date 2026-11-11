package storage_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/storage"
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

// A pool of connections must not turn concurrent writes into SQLITE_BUSY. Two
// things make that work and both are added to the DSN by OpenPooled: a
// busy_timeout, so a competing write waits, and _txlock=immediate, so a
// transaction takes the write lock upfront instead of failing on the upgrade
// (a deferred transaction that starts as a reader gets SQLITE_BUSY immediately,
// without waiting out the timeout).
func TestOpenPooled_ConcurrentWritesAndReads(t *testing.T) {
	dsn := "file:" + filepath.Join(t.TempDir(), "pool.db") + "?_pragma=journal_mode(WAL)"
	db, err := storage.OpenPooled(dsn, 4)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatal(err)
	}

	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	const writers, perWriter = 4, 25
	errs := make(chan error, writers*perWriter)
	var wg sync.WaitGroup

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				uuid := "w" + strconv.Itoa(w) + "-" + strconv.Itoa(i)
				if err := store.Store(ctx, makeEvent(uuid, "sentry", "default")); err != nil {
					errs <- err
					return
				}
				// A reader running against the same pool while writes are in flight.
				if _, err := store.FindAll(ctx, event.FindOptions{Project: "default", Limit: 10}); err != nil {
					errs <- err
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent access failed: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != writers*perWriter {
		t.Fatalf("stored %d events, want %d", count, writers*perWriter)
	}
}

// A caller that asks for one connection (or a nonsensical number) keeps the
// historical behaviour and an untouched DSN.
func TestOpenPooled_SingleConnection(t *testing.T) {
	for _, maxOpen := range []int{0, 1} {
		db, err := storage.OpenPooled(":memory:", maxOpen)
		if err != nil {
			t.Fatalf("maxOpen=%d: %v", maxOpen, err)
		}
		if got := db.Stats().MaxOpenConnections; got != 1 {
			t.Errorf("maxOpen=%d: MaxOpenConnections = %d, want 1", maxOpen, got)
		}
		db.Close()
	}
}

// In-memory databases share one cache between connections, and shared-cache
// mode uses table locks that busy_timeout does not cover: a pool fails with
// "database table is locked" under concurrent writes. OpenPooled must keep
// in-memory DSNs on one connection, whatever pool size is asked for.
func TestOpenPooled_InMemoryUsesOneConnection(t *testing.T) {
	for _, dsn := range []string{":memory:", "file::memory:?cache=shared", "file:mem?mode=memory&cache=shared"} {
		t.Run(dsn, func(t *testing.T) {
			db, err := storage.OpenPooled(dsn, 4)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			if got := db.Stats().MaxOpenConnections; got != 1 {
				t.Errorf("MaxOpenConnections = %d, want 1", got)
			}

			if _, err := db.Exec(`CREATE TABLE events (
				uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
				timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
			)`); err != nil {
				t.Fatal(err)
			}
			store := storage.NewSQLiteStore(db)
			ctx := context.Background()

			const workers, perWorker = 8, 50
			errs := make(chan error, workers)
			var wg sync.WaitGroup
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func(w int) {
					defer wg.Done()
					for i := 0; i < perWorker; i++ {
						uuid := "w" + strconv.Itoa(w) + "-" + strconv.Itoa(i)
						tx, err := db.Begin()
						if err != nil {
							errs <- err
							return
						}
						if _, err := tx.Exec(`INSERT INTO events (uuid, type, payload, timestamp) VALUES (?, 'tx', '{}', '0')`, "tx-"+uuid); err != nil {
							tx.Rollback()
							errs <- err
							return
						}
						if err := tx.Commit(); err != nil {
							errs <- err
							return
						}
						if err := store.Store(ctx, makeEvent(uuid, "sentry", "default")); err != nil {
							errs <- err
							return
						}
						if _, err := store.FindAll(ctx, event.FindOptions{Project: "default", Limit: 10}); err != nil {
							errs <- err
							return
						}
					}
				}(w)
			}
			wg.Wait()
			close(errs)
			for err := range errs {
				t.Fatalf("concurrent access failed: %v", err)
			}
		})
	}
}

// Clear with pinned events used to send the ids of the loaded unpinned events
// (at most 500); the server now deletes every match except the pinned ones.
func TestSQLiteStore_DeleteAll_KeepPinned(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	for i := 0; i < 600; i++ {
		store.Store(ctx, makeEvent("s"+strconv.Itoa(i), "sentry", "a"))
	}
	store.Store(ctx, makeEvent("other-project", "sentry", "b"))
	store.Store(ctx, makeEvent("other-type", "ray", "a"))
	store.Pin(ctx, "s7")
	store.Pin(ctx, "s599")

	if err := store.DeleteAll(ctx, event.DeleteOptions{Type: "sentry", Project: "a", KeepPinned: true}); err != nil {
		t.Fatal(err)
	}
	all, _ := store.FindAll(ctx, event.FindOptions{Limit: 1000})
	got := map[string]bool{}
	for _, ev := range all {
		got[ev.UUID] = true
	}
	want := []string{"s7", "s599", "other-project", "other-type"}
	if len(got) != len(want) {
		t.Errorf("left %d events %v, want %v", len(got), got, want)
	}
	for _, u := range want {
		if !got[u] {
			t.Errorf("%s was deleted", u)
		}
	}

	// keep_pinned with ids keeps a pinned id too.
	if err := store.DeleteAll(ctx, event.DeleteOptions{UUIDs: []string{"s7", "other-type"}, KeepPinned: true}); err != nil {
		t.Fatal(err)
	}
	if ev, _ := store.FindByUUID(ctx, "s7"); ev == nil {
		t.Error("pinned s7 deleted by id with keep_pinned")
	}
	if ev, _ := store.FindByUUID(ctx, "other-type"); ev != nil {
		t.Error("unpinned other-type kept")
	}
}

func TestSQLiteStore_OnDelete(t *testing.T) {
	db := setupTestDB(t)
	store := storage.NewSQLiteStore(db)
	ctx := context.Background()

	var calls [][]string
	store.OnDelete(func(uuids []string) {
		calls = append(calls, append([]string(nil), uuids...))
	})
	take := func() []string {
		t.Helper()
		if len(calls) > 1 {
			t.Fatalf("hook called %d times, want at most 1", len(calls))
		}
		var got []string
		if len(calls) == 1 {
			got = calls[0]
			sort.Strings(got)
		}
		calls = nil
		return got
	}

	for _, ev := range []event.Event{
		makeEvent("h1", "sentry", "a"),
		makeEvent("h2", "sentry", "a"),
		makeEvent("h3", "sentry", "b"),
		makeEvent("h4", "ray", "a"),
		makeEvent("h5", "ray", "a"),
	} {
		store.Store(ctx, ev)
	}
	store.Pin(ctx, "h5")

	steps := []struct {
		name string
		run  func() error
		want []string
	}{
		{"single", func() error { return store.Delete(ctx, "h1") }, []string{"h1"}},
		{"missing uuid", func() error { return store.Delete(ctx, "nope") }, nil},
		{"ids keep pinned", func() error {
			return store.DeleteAll(ctx, event.DeleteOptions{UUIDs: []string{"h4", "h5"}, KeepPinned: true})
		}, []string{"h4"}},
		{"type and project", func() error {
			return store.DeleteAll(ctx, event.DeleteOptions{Type: "sentry", Project: "a"})
		}, []string{"h2"}},
		{"all", func() error { return store.DeleteAll(ctx, event.DeleteOptions{}) }, []string{"h3", "h5"}},
		{"empty table", func() error { return store.DeleteAll(ctx, event.DeleteOptions{}) }, nil},
	}
	for _, st := range steps {
		if err := st.run(); err != nil {
			t.Fatalf("%s: %v", st.name, err)
		}
		if got := take(); !reflect.DeepEqual(got, st.want) {
			t.Errorf("%s: hook got %v, want %v", st.name, got, st.want)
		}
	}
}
