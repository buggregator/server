package ray

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/module"
	serverhttp "github.com/buggregator/go-buggregator/internal/server/http"
	"github.com/buggregator/go-buggregator/internal/server/ws"
	"github.com/buggregator/go-buggregator/internal/storage"
)

func newRayService(t *testing.T) (*serverhttp.EventService, *storage.SQLiteStore) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	store := storage.NewSQLiteStore(db)
	return serverhttp.NewEventService(store, ws.NewHub(), module.NewRegistry(), nil, db), store
}

func sendRay(t *testing.T, es *serverhttp.EventService, body string) error {
	t.Helper()
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	inc, err := (&handler{}).Handle(r)
	if err != nil {
		t.Fatal(err)
	}
	return es.HandleIncoming(context.Background(), inc)
}

func rayPayloadTypes(t *testing.T, store *storage.SQLiteStore, uuid string) []string {
	t.Helper()
	ev, err := store.FindByUUID(context.Background(), uuid)
	if err != nil || ev == nil {
		t.Fatalf("event %s: %v", uuid, err)
	}
	var doc struct {
		Payloads []struct {
			Type string `json:"type"`
		} `json:"payloads"`
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(ev.Payload, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Meta["php_version"] != "8.3" {
		t.Errorf("meta lost: %s", ev.Payload)
	}
	var types []string
	for _, p := range doc.Payloads {
		types = append(types, p.Type)
	}
	return types
}

// ray('test')->trace() sends the trace as a second request with the first
// one's uuid: it joins the stored event instead of failing on UNIQUE (#345).
func TestRay_FollowUpMergesIntoEvent(t *testing.T) {
	es, store := newRayService(t)
	const first = `{"uuid":"u-1","payloads":[{"type":"log","content":{"values":["test"]}}],"meta":{"php_version":"8.3"}}`
	const trace = `{"uuid":"u-1","payloads":[{"type":"trace","content":{"frames":[]}}],"meta":{"php_version":"8.3"}}`
	const color = `{"uuid":"u-1","payloads":[{"type":"color","content":{"color":"red"}}],"meta":{"php_version":"8.3"}}`
	for _, body := range []string{first, trace, color} {
		if err := sendRay(t, es, body); err != nil {
			t.Fatalf("send: %v", err)
		}
	}
	if got := strings.Join(rayPayloadTypes(t, store, "u-1"), ","); got != "log,trace,color" {
		t.Errorf("payloads = %s", got)
	}
	all, _ := store.FindAll(context.Background(), event.FindOptions{})
	if len(all) != 1 {
		t.Errorf("%d events, want 1", len(all))
	}
}

// Follow-ups racing the first payload all land in one event.
func TestRay_ConcurrentFollowUps(t *testing.T) {
	es, store := newRayService(t)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- sendRay(t, es, fmt.Sprintf(`{"uuid":"u-2","payloads":[{"type":"p%d"}],"meta":{"php_version":"8.3"}}`, i))
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("send: %v", err)
		}
	}
	if n := len(rayPayloadTypes(t, store, "u-2")); n != 20 {
		t.Errorf("%d payloads, want 20", n)
	}
}
