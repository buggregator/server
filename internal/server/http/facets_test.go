package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	serverhttp "github.com/buggregator/server/v2/internal/server/http"
	"github.com/buggregator/server/v2/internal/server/ws"
	"github.com/buggregator/server/v2/internal/storage"
)

// setupFacetAPI: the API over a migrated file database whose store indexes
// {"host","env","tag"} of test payloads.
func setupFacetAPI(t *testing.T) (*http.ServeMux, *storage.SQLiteStore) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	registry := module.NewRegistry()
	mux := http.NewServeMux()
	store := storage.NewSQLiteStore(db)
	if err := registry.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}
	if err := storage.InitFacets(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	store.SetFacetFunc(func(_ string, payload json.RawMessage) []event.Facet {
		var p struct{ Host, Env, Feature string }
		_ = json.Unmarshal(payload, &p)
		var s event.FacetSet
		s.Add(event.FacetHost, p.Host)
		s.Add(event.FacetEnv, p.Env)
		s.AddTag("feature", p.Feature)
		return s.List()
	})
	es := serverhttp.NewEventService(store, ws.NewHub(), registry, nil, db)
	noop := func(next http.Handler) http.Handler { return next }
	serverhttp.RegisterAPI(mux, store, event.NewPreviewRegistry(), es, "test", db, []string{"sentry"},
		serverhttp.AuthSettings{}, noop, serverhttp.ListLimits{}, nil, map[string]int{"http": 8000})
	return mux, store
}

func getJSON(t *testing.T, mux *http.ServeMux, url string, out any) {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("%s: status %d: %s", url, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatal(err)
	}
}

func TestAPI_FeedFilters(t *testing.T) {
	mux, store := setupFacetAPI(t)
	for i, p := range []string{
		`{"host":"web-1","env":"dev","feature":"cart"}`,
		`{"host":"web-2","env":"dev"}`,
		`{"host":"web-1","env":"prod"}`,
		`{}`,
	} {
		ev := event.Event{UUID: string(rune('a' + i)), Type: "sentry", Payload: json.RawMessage(p), Timestamp: float64(i + 1)}
		if err := store.Store(context.Background(), ev); err != nil {
			t.Fatal(err)
		}
	}

	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"host=web-1", []string{"a", "c"}},
		{"host=web-1&host=web-2&env=dev", []string{"a", "b"}},
		{"not.host=web-1", []string{"b", "d"}},
		{"tag=feature:cart", []string{"a"}},
		{"tag=feature", []string{"a", "b", "c", "d"}}, // no value: ignored
		{"bogus=1&host=", []string{"a", "b", "c", "d"}},
	} {
		for _, path := range []string{"/api/events", "/api/events/preview"} {
			var resp struct {
				Data []struct {
					UUID string `json:"uuid"`
				} `json:"data"`
			}
			getJSON(t, mux, path+"?"+tc.query, &resp)
			var got []string
			for _, e := range resp.Data {
				got = append(got, e.UUID)
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("%s?%s = %v, want %v", path, tc.query, got, tc.want)
			}
		}
	}

	var facets struct {
		Data map[string][]event.FacetCount `json:"data"`
		Keys []string                      `json:"keys"`
	}
	getJSON(t, mux, "/api/events/facets?env=dev", &facets)
	want := map[string][]event.FacetCount{
		"host":        {{Value: "web-1", Count: 1}, {Value: "web-2", Count: 1}},
		"env":         {{Value: "dev", Count: 2}, {Value: "prod", Count: 1}}, // own filter ignored
		"tag:feature": {{Value: "cart", Count: 1}},
	}
	if !reflect.DeepEqual(facets.Data, want) {
		t.Errorf("facets = %v, want %v", facets.Data, want)
	}
	if !reflect.DeepEqual(facets.Keys, []string{"env", "host", "tag:feature"}) {
		t.Errorf("keys = %v", facets.Keys)
	}
}
