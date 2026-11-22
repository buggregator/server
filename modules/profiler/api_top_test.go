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

// GET /api/profiler/{uuid}/top: order=asc sorts the whole profile before the
// limit (the lowest N, not the top N reversed) and total counts all functions.
func TestTopFunctions_OrderAndTotal(t *testing.T) {
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

	// main() calls f00..f14; fNN takes (NN+1)*100 µs of CPU.
	var b strings.Builder
	b.WriteString(`{"app_name":"app","hostname":"h","date":1711800000,"profile":{"main()":{"wt":100000,"cpu":100000,"mu":1,"pmu":1,"ct":1}`)
	for i := 0; i < 15; i++ {
		name := "f" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		cpu := (i + 1) * 100
		b.WriteString(`,"main()==>` + name + `()":{"wt":` + itoa(cpu) + `,"cpu":` + itoa(cpu) + `,"mu":1,"pmu":1,"ct":1}`)
	}
	b.WriteString(`}}`)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(b.String()))
	req.Header.Set("X-Profiler-Dump", "1")
	req.Header.Set("Content-Type", "application/json")
	var uuid string
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
			uuid = ev.UUID
		}
	}
	if uuid == "" {
		t.Fatal("profile not ingested")
	}

	top := func(query string) (names []string, total int) {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/profiler/"+uuid+"/top?"+query, nil))
		var resp struct {
			Total     int              `json:"total"`
			Functions []map[string]any `json:"functions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("%s: %v (%s)", query, err, rec.Body.String())
		}
		for _, f := range resp.Functions {
			names = append(names, f["function"].(string))
		}
		return names, resp.Total
	}

	desc, total := top("limit=10&metric=excl_cpu")
	if total != 16 { // main() + 15
		t.Errorf("total = %d, want 16", total)
	}
	if len(desc) != 10 || desc[0] != "main()" || desc[1] != "f14()" {
		t.Errorf("desc = %v", desc)
	}
	asc, total := top("limit=10&metric=excl_cpu&order=asc")
	if total != 16 {
		t.Errorf("asc total = %d, want 16", total)
	}
	if len(asc) != 10 || asc[0] != "f00()" || asc[9] != "f09()" {
		t.Errorf("asc = %v, want f00()..f09()", asc)
	}
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }
