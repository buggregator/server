package profiler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/modules/profiler"
)

// A recursive profile, XHProf-style: foo recurses two levels deep, calls bar
// from levels 0 and 1, and baz at levels 0 and 1 (baz@1 nested in baz).
const recursiveProfile = `{"app_name":"app","hostname":"h","date":1711800000,"profile":{
	"main()":{"wt":1000,"cpu":1000,"mu":1000,"pmu":1000,"ct":1},
	"main()==>foo":{"wt":800,"cpu":800,"mu":800,"pmu":800,"ct":1},
	"foo==>foo@1":{"wt":600,"cpu":600,"mu":600,"pmu":600,"ct":1},
	"foo@1==>foo@2":{"wt":300,"cpu":300,"mu":300,"pmu":300,"ct":1},
	"foo==>bar":{"wt":100,"cpu":100,"mu":100,"pmu":100,"ct":1},
	"foo@1==>bar":{"wt":100,"cpu":100,"mu":100,"pmu":100,"ct":1},
	"foo==>baz":{"wt":50,"cpu":50,"mu":50,"pmu":50,"ct":1},
	"foo@1==>baz@1":{"wt":20,"cpu":20,"mu":20,"pmu":20,"ct":1},
	"main()==>user@example":{"wt":10,"cpu":10,"mu":10,"pmu":10,"ct":1}}}`

type graphResp struct {
	Recursion string `json:"recursion"`
	Nodes     []struct {
		Data struct {
			ID   string           `json:"id"`
			Name string           `json:"name"`
			Cost map[string]int64 `json:"cost"`
		} `json:"data"`
	} `json:"nodes"`
	Edges []struct {
		Data struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Label  string `json:"label"`
		} `json:"data"`
	} `json:"edges"`
}

func ingestProfile(t *testing.T, body string) (*http.ServeMux, string) {
	t.Helper()
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

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
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
			return mux, ev.UUID
		}
	}
	t.Fatal("profile not ingested")
	return nil, ""
}

func callGraph(t *testing.T, mux *http.ServeMux, uuid, query string) graphResp {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/profiler/"+uuid+"/call-graph?"+query, nil))
	var g graphResp
	if err := json.Unmarshal(rec.Body.Bytes(), &g); err != nil {
		t.Fatalf("%s: %v (%s)", query, err, rec.Body.String())
	}
	return g
}

// edges lists the graph's calls as "caller -> callee label", by node names
// without the call counts.
func (g graphResp) edges() []string {
	names := map[string]string{}
	for _, n := range g.Nodes {
		name := n.Data.Name
		if i := strings.Index(name, " ("); i > 0 {
			name = name[:i]
		}
		names[n.Data.ID] = name
	}
	var out []string
	for _, e := range g.Edges {
		out = append(out, names[e.Data.Source]+" -> "+names[e.Data.Target]+" "+e.Data.Label)
	}
	sort.Strings(out)
	return out
}

func (g graphResp) nodeNames() []string {
	var out []string
	for _, n := range g.Nodes {
		out = append(out, n.Data.Name)
	}
	sort.Strings(out)
	return out
}

func TestCallGraph_Recursion(t *testing.T) {
	mux, uuid := ingestProfile(t, recursiveProfile)
	q := "metric=wt&threshold=0&percentage=0"

	levels := callGraph(t, mux, uuid, q)
	if levels.Recursion != "levels" {
		t.Errorf("recursion = %q, want levels", levels.Recursion)
	}
	if got := levels.nodeNames(); len(got) != 8 || !contains(got, "foo@2 (1x)") || !contains(got, "baz@1 (1x)") {
		t.Errorf("levels nodes = %v", got)
	}
	// Any other value keeps the levels.
	if g := callGraph(t, mux, uuid, q+"&recursion=bogus"); g.Recursion != "levels" || len(g.Nodes) != 8 {
		t.Errorf("bogus recursion: %q, %d nodes", g.Recursion, len(g.Nodes))
	}

	g := callGraph(t, mux, uuid, q+"&recursion=collapsed")
	if g.Recursion != "collapsed" {
		t.Errorf("recursion = %q, want collapsed", g.Recursion)
	}
	wantNodes := []string{"bar (1x)", "baz (2x, 1 recursive)", "foo (3x, 2 recursive)", "main() (1x)", "user@example (1x)"}
	if got := g.nodeNames(); strings.Join(got, "|") != strings.Join(wantNodes, "|") {
		t.Errorf("nodes = %v\nwant    %v", got, wantNodes)
	}
	for _, n := range g.Nodes {
		if strings.HasPrefix(n.Data.Name, "foo ") && (n.Data.Cost["wt"] != 800 || n.Data.Cost["ct"] != 3) {
			t.Errorf("foo cost = %v, want the outermost wt 800 and ct 3", n.Data.Cost)
		}
	}
	// foo→foo: foo@1→foo@2 is nested in foo→foo@1, so only its count adds;
	// foo@1→bar is a call apart (bar is not deeper), so it adds in full.
	wantEdges := []string{
		"foo -> bar 20.00%",
		"foo -> baz 5.00%",
		"foo -> foo 60.00%",
		"main() -> foo 80.00%",
		"main() -> user@example 1.00%",
	}
	if got := g.edges(); strings.Join(got, "|") != strings.Join(wantEdges, "|") {
		t.Errorf("edges = %v\nwant    %v", got, wantEdges)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
