package profiler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/profiler"
)

// GET /api/profiler/compare: the shape profile-compare.vue reads
// (functions[].base_/compare_/diff_ excl_wt, excl_mu, ct), biggest
// wall-time change first, functions of either profile.
func TestCompare(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := storage.NewSQLiteStore(db)
	mux := http.NewServeMux()
	reg := module.NewRegistry()
	mod := profiler.New()
	reg.Register(mod)
	if err := reg.Init(db, mux, store); err != nil {
		t.Fatal(err)
	}
	mod.RegisterProtectedRoutes(mux, func(h http.Handler) http.Handler { return h })

	ingest := func(profile string) string {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"app_name":"app","hostname":"h","date":1711800000,"profile":{`+profile+`}}`))
		req.Header.Set("X-Profiler-Dump", "1")
		req.Header.Set("Content-Type", "application/json")
		for _, h := range reg.Handlers() {
			if h.Match(req) {
				inc, err := h.Handle(req)
				if err != nil {
					t.Fatal(err)
				}
				ev := event.NewEvent(inc)
				if err := store.Store(context.Background(), ev); err != nil {
					t.Fatal(err)
				}
				reg.NotifyEventStored(ev)
				return ev.UUID
			}
		}
		t.Fatal("profile not ingested")
		return ""
	}
	// Before: a() 1000 µs, b() 500 µs. After: a() slower and called twice,
	// b() gone, c() new.
	base := ingest(`"main()":{"wt":2000,"cpu":2000,"mu":1000,"pmu":1000,"ct":1},
		"main()==>a()":{"wt":1000,"cpu":1000,"mu":100,"pmu":100,"ct":1},
		"main()==>b()":{"wt":500,"cpu":500,"mu":50,"pmu":50,"ct":1}`)
	after := ingest(`"main()":{"wt":3500,"cpu":3500,"mu":1000,"pmu":1000,"ct":1},
		"main()==>a()":{"wt":3000,"cpu":3000,"mu":300,"pmu":300,"ct":2},
		"main()==>c()":{"wt":200,"cpu":200,"mu":20,"pmu":20,"ct":1}`)

	get := func(query string) (int, map[string]any) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/profiler/compare?"+query, nil))
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return rec.Code, resp
	}

	code, resp := get("base=" + base + "&compare=" + after)
	if code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	fns := resp["functions"].([]any)
	byName := map[string]map[string]any{}
	var order []string
	for _, f := range fns {
		m := f.(map[string]any)
		byName[m["function"].(string)] = m
		order = append(order, m["function"].(string))
	}
	if len(fns) != 4 || resp["total"].(float64) != 4 || order[0] != "a()" {
		t.Fatalf("functions %v, total %v; want 4, a() first", order, resp["total"])
	}
	for name, want := range map[string][9]float64{
		"a()": {1000, 3000, 2000, 100, 300, 200, 1, 2, 1},
		"b()": {500, 0, -500, 50, 0, -50, 1, 0, -1},
		"c()": {0, 200, 200, 0, 20, 20, 0, 1, 1},
	} {
		f := byName[name]
		got := [9]float64{}
		for i, k := range []string{"base_excl_wt", "compare_excl_wt", "diff_excl_wt", "base_excl_mu", "compare_excl_mu", "diff_excl_mu", "base_ct", "compare_ct", "diff_ct"} {
			got[i], _ = f[k].(float64)
		}
		if got != want {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	if bt := resp["base_totals"].(map[string]any); bt["wt"].(float64) != 2000 {
		t.Errorf("base_totals = %v", bt)
	}

	if _, resp := get("base=" + base + "&compare=" + after + "&limit=2"); len(resp["functions"].([]any)) != 2 || resp["total"].(float64) != 4 {
		t.Errorf("limit=2: %v", resp)
	}
	for query, want := range map[string]int{
		"base=" + base: http.StatusBadRequest,
		"base=" + base + "&compare=" + after + "&limit=0": http.StatusBadRequest,
		"base=" + base + "&compare=nope":                  http.StatusNotFound,
		"base=nope&compare=" + after:                      http.StatusNotFound,
	} {
		if code, _ := get(query); code != want {
			t.Errorf("%s: status %d, want %d", query, code, want)
		}
	}
}
