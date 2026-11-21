package codegraph

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/storage"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// copyTree copies testdata/app into a temp dir so tests may change files.
func copyTree(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.Walk("testdata/app", func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel("testdata/app", p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("code-graph", migrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	return db
}

func setup(t *testing.T, cfg Config) (*Module, string) {
	t.Helper()
	dir := copyTree(t)
	if cfg.Roots == nil {
		cfg.Roots = []Root{{Name: "app", Path: dir}}
	}
	db := openDB(t)
	m := New(cfg)
	if err := m.OnInit(db); err != nil {
		t.Fatal(err)
	}
	m.Indexer().Wait()
	return m, dir
}

func mustLookup(t *testing.T, m *Module, fqn string) SymbolRef {
	t.Helper()
	s, err := m.graphs.Lookup(fqn, "")
	if err != nil {
		t.Fatalf("lookup %s: %v", fqn, err)
	}
	return s
}

type edgeKey struct{ to, kind, confidence string }

// calls returns the edges of Calls keyed by the other side's FQN.
func calls(t *testing.T, m *Module, fqn string, incoming bool) map[edgeKey]bool {
	t.Helper()
	s := mustLookup(t, m, fqn)
	g, err := m.graphs.Calls(s.ID, incoming, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]GraphNode{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	out := map[edgeKey]bool{}
	for _, e := range g.Edges {
		other := e.To
		if incoming {
			other = e.From
		}
		n := byID[other]
		name := n.FQN
		if n.External {
			name = "external:" + n.FQN
		}
		out[edgeKey{name, e.Kind, e.Confidence}] = true
	}
	return out
}

func TestIndex_ResolvesCalls(t *testing.T) {
	m, _ := setup(t, Config{})
	st, _ := m.ix.Status("app")
	if st.State != StateReady || st.Error != "" {
		t.Fatalf("status %+v", st)
	}
	if st.Files != 8 { // 4 PHP + 4 Go; vendor/ excluded by default
		t.Fatalf("files = %d, want 8", st.Files)
	}

	tests := []struct {
		name     string
		symbol   string
		incoming bool
		want     []edgeKey
		absent   []edgeKey
	}{
		{"php static, self, parent, $this, inherited", `App\Service\UserService::save`, false, []edgeKey{
			{`App\Repo\UserRepo::find`, EdgeCalls, ConfExact},
			{`App\Service\UserService::log`, EdgeCalls, ConfExact},
			{`App\Service\UserService::make`, EdgeCalls, ConfExact},
			{`App\Service\Base::boot`, EdgeCalls, ConfExact}, // parent::boot() and inherited $this->boot()
			{`external:persist`, EdgeCalls, ConfName},        // $this->repo->persist(): type unknown
			{`external:App\Model\Post`, EdgeInstantiates, ConfExact},
			{`external:App\Service\strlen`, EdgeCalls, ConfExact},
		}, nil},
		{"php function in namespace", `App\Service\Base::boot`, false, []edgeKey{
			{`App\Service\helper2`, EdgeCalls, ConfExact},
		}, nil},
		{"php callers by name", `App\Repo\UserRepo::persist`, true, []edgeKey{
			{`App\Service\UserService::save`, EdgeCalls, ConfName},
		}, nil},
		{"php instantiation", `App\Repo\UserRepo`, true, []edgeKey{
			{`App\Service\top`, EdgeInstantiates, ConfExact},
		}, nil},
		{"go package call, local call", `example.com/app/cmd.main`, false, []edgeKey{
			{`example.com/app/internal/repo.New`, EdgeCalls, ConfExact},
			{`example.com/app/cmd.run`, EdgeCalls, ConfExact},
			{`external:fmt.Println`, EdgeCalls, ConfExact},
			{`external:Persist`, EdgeCalls, ConfName},
		}, nil},
		{"go receiver call across files", `example.com/app/internal/repo.Repo.Persist`, false, []edgeKey{
			{`example.com/app/internal/repo.Repo.flush`, EdgeCalls, ConfExact},
		}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calls(t, m, tc.symbol, tc.incoming)
			for _, w := range tc.want {
				if !got[w] {
					t.Errorf("missing %+v; got %v", w, got)
				}
			}
		})
	}

	// A Go method in another file than its type hangs under the type.
	flush := mustLookup(t, m, "example.com/app/internal/repo.Repo.flush")
	repo := mustLookup(t, m, "example.com/app/internal/repo.Repo")
	if flush.ParentID != repo.ID {
		t.Errorf("flush parent = %d, want Repo %d", flush.ParentID, repo.ID)
	}

	d, err := m.graphs.Detail(mustLookup(t, m, `App\Service\UserService`).ID)
	if err != nil {
		t.Fatal(err)
	}
	rel := map[string]bool{}
	for _, r := range d.Relations {
		rel[r.Kind+" "+r.FQN+" "+boolStr(r.Symbol != nil)] = true
	}
	for _, w := range []string{`extends App\Service\Base true`, `implements App\Service\Saver true`, `implements JsonSerializable false`, `uses App\Service\T true`} {
		if !rel[w] {
			t.Errorf("relation %q missing: %v", w, rel)
		}
	}
	if d.Source != nil {
		t.Error("source served with snippets off")
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestIndex_Incremental(t *testing.T) {
	m, dir := setup(t, Config{})
	run := func() Status {
		t.Helper()
		m.ix.Reindex("app")
		m.ix.Wait()
		st, _ := m.ix.Status("app")
		if st.State != StateReady {
			t.Fatalf("status %+v", st)
		}
		return st
	}

	if st := run(); st.FilesParsed != 0 {
		t.Fatalf("unchanged tree parsed %d files", st.FilesParsed)
	}

	// Same content, new mtime: hashed, not parsed.
	repoFile := filepath.Join(dir, "src/Repo/UserRepo.php")
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(repoFile, later, later); err != nil {
		t.Fatal(err)
	}
	if st := run(); st.FilesParsed != 0 {
		t.Fatalf("touched file parsed: %d", st.FilesParsed)
	}

	// Changed file: parsed again; edges to it resolve again.
	src, _ := os.ReadFile(repoFile)
	if err := os.WriteFile(repoFile, []byte(strings.Replace(string(src), "class UserRepo", "class UserRepo\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if st := run(); st.FilesParsed != 1 {
		t.Fatalf("changed file: parsed %d, want 1", st.FilesParsed)
	}
	if got := calls(t, m, `App\Service\UserService::save`, false); !got[edgeKey{`App\Repo\UserRepo::find`, EdgeCalls, ConfExact}] {
		t.Errorf("edge to the re-parsed file lost: %v", got)
	}

	// Deleted file: its rows go, the edge to it becomes external.
	if err := os.Remove(repoFile); err != nil {
		t.Fatal(err)
	}
	st := run()
	if st.Files != 7 {
		t.Fatalf("files = %d after delete, want 7", st.Files)
	}
	if _, err := m.graphs.Lookup(`App\Repo\UserRepo`, ""); err == nil {
		t.Error("symbol of a deleted file still found")
	}
	if got := calls(t, m, `App\Service\UserService::save`, false); !got[edgeKey{`external:App\Repo\UserRepo::find`, EdgeCalls, ConfExact}] {
		t.Errorf("edge to a deleted file not external: %v", got)
	}
}

func TestIndex_RootsFromConfigOnly(t *testing.T) {
	db := openDB(t)
	dir := copyTree(t)
	m := New(Config{Roots: []Root{{Name: "app", Path: dir}}})
	if err := m.OnInit(db); err != nil {
		t.Fatal(err)
	}
	m.Indexer().Wait()

	// The root leaves the config: its rows go at the next start.
	other := t.TempDir()
	m2 := New(Config{Roots: []Root{{Name: "other", Path: other}, {Name: "missing", Path: filepath.Join(other, "nope")}}})
	if err := m2.OnInit(db); err != nil {
		t.Fatal(err)
	}
	m2.Indexer().Wait()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM code_symbols WHERE root = 'app'`).Scan(&n)
	if n != 0 {
		t.Errorf("%d symbols of a removed root left", n)
	}
	if roots := m2.Roots(); len(roots) != 1 || roots[0].Name != "other" {
		t.Errorf("roots = %+v, want only the existing one", roots)
	}
}

func TestSubgraph(t *testing.T) {
	m, _ := setup(t, Config{})
	hasEdge := func(g *Graph, from, to, kind string) bool {
		for _, e := range g.Edges {
			if e.From == from && e.To == to && e.Kind == kind {
				return true
			}
		}
		return false
	}

	g, err := m.graphs.Subgraph(GraphQuery{Root: "app"})
	if err != nil {
		t.Fatal(err)
	}
	if !hasEdge(g, `ns:App\Service`, `ns:App\Repo`, EdgeCalls) || !hasEdge(g, "ns:example.com/app/cmd", "ns:example.com/app/internal/repo", EdgeImports) {
		t.Errorf("package graph: %+v", g.Edges)
	}

	svc := mustLookup(t, m, `App\Service\UserService`)
	base := mustLookup(t, m, `App\Service\Base`)
	g, err = m.graphs.Subgraph(GraphQuery{Root: "app", Level: LevelClass, Target: svc.ID, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	sid := func(s SymbolRef) string { return "s:" + itoa(s.ID) }
	if !hasEdge(g, sid(svc), sid(base), EdgeExtends) || !hasEdge(g, sid(svc), sid(base), EdgeCalls) {
		t.Errorf("class graph: %+v", g.Edges)
	}

	g, err = m.graphs.Subgraph(GraphQuery{Root: "app", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !g.Truncated || len(g.Nodes) != 1 || len(g.Edges) != 0 {
		t.Errorf("limit 1: truncated=%v nodes=%d edges=%d", g.Truncated, len(g.Nodes), len(g.Edges))
	}
}

func itoa(i int64) string { b, _ := json.Marshal(i); return string(b) }

func TestResolveFrame(t *testing.T) {
	m, dir := setup(t, Config{PathMap: []PathMap{{From: "/var/www/html", To: "PLACEHOLDER"}}})
	m.graphs.pathMap[0].To = m.cfg.Roots[0].Path
	_ = dir
	tests := []struct {
		file string
		line int
		want string
	}{
		{"/var/www/html/src/Service/UserService.php", 20, `App\Service\UserService::save`},
		{"src/Service/UserService.php", 14, `App\Service\UserService`},
		{filepath.Join(m.cfg.Roots[0].Path, "go/cmd/main.go"), 11, "example.com/app/cmd.main"},
	}
	for _, tc := range tests {
		s, err := m.graphs.Resolve(tc.file, tc.line, "")
		if err != nil {
			t.Errorf("%s:%d: %v", tc.file, tc.line, err)
			continue
		}
		if s.FQN != tc.want {
			t.Errorf("%s:%d = %s, want %s", tc.file, tc.line, s.FQN, tc.want)
		}
	}
	if _, err := m.graphs.Resolve("/etc/passwd", 1, ""); err == nil {
		t.Error("a file outside every root resolved")
	}
}

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok.php"), []byte("<?php"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.php"), filepath.Join(root, "link.php")); err != nil {
		t.Skip("symlinks not available:", err)
	}
	r := Root{Name: "r", Path: root}
	for _, tc := range []struct {
		rel string
		ok  bool
	}{
		{"ok.php", true},
		{"link.php", false},
		{"../" + filepath.Base(outside) + "/secret.php", false},
	} {
		_, err := safeJoin(r, tc.rel)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err=%v, want ok=%v", tc.rel, err, tc.ok)
		}
	}
}

func TestAPI(t *testing.T) {
	m, _ := setup(t, Config{Snippets: true})
	mux := http.NewServeMux()
	m.RegisterProtectedRoutes(mux, func(h http.Handler) http.Handler { return h })
	get := func(method, url string, wantStatus int) map[string]any {
		t.Helper()
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(method, url, nil))
		if rec.Code != wantStatus {
			t.Fatalf("%s %s: %d, want %d: %s", method, url, rec.Code, wantStatus, rec.Body)
		}
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return out
	}

	roots := get("GET", "/api/code/roots", 200)
	if roots["enabled"] != true || len(roots["roots"].([]any)) != 1 {
		t.Errorf("roots: %v", roots)
	}
	get("GET", "/api/code/roots/nope", 404)
	get("POST", "/api/code/roots/app/reindex", 202)
	m.ix.Wait()

	if top := get("GET", "/api/code/top", 200); len(top["symbols"].([]any)) == 0 {
		t.Errorf("top: %v", top)
	}
	get("GET", "/api/code/graph?expand=ns:App%5CService&expand=s:1", 200)
	res := get("GET", "/api/code/search?q=UserRepo&kind=class", 200)
	syms := res["symbols"].([]any)
	if len(syms) == 0 || syms[0].(map[string]any)["fqn"] != `App\Repo\UserRepo` {
		t.Fatalf("search: %v", res)
	}
	id := int64(syms[0].(map[string]any)["id"].(float64))
	d := get("GET", "/api/code/symbols/"+itoa(id), 200)
	if src, _ := d["source"].(string); !strings.Contains(src, "class UserRepo") {
		t.Errorf("snippet: %v", d["source"])
	}
	get("GET", "/api/code/symbols/"+itoa(id)+"/callers?depth=2", 200)
	get("GET", "/api/code/symbols/"+itoa(id)+"/callees", 200)
	get("GET", "/api/code/symbols/abc", 400)
	get("GET", "/api/code/symbols/999999", 404)

	files := get("GET", "/api/code/files?path=src", 200)
	if n := len(files["entries"].([]any)); n != 2 { // Repo/, Service/
		t.Errorf("files under src: %v", files)
	}
	get("GET", "/api/code/files/outline?path=src/Service/UserService.php", 200)
	get("GET", "/api/code/files/outline?path=../../etc/passwd", 404)
	get("GET", "/api/code/graph?level=class&target="+itoa(id), 200)
	get("GET", `/api/code/graph?target=App%5CRepo%5CUserRepo%3A%3Afind&direction=in`, 200)
	get("GET", "/api/code/graph?level=bogus", 400)
	get("GET", "/api/code/graph?root=nope", 404)
	get("GET", "/api/code/resolve?file=src/Repo/UserRepo.php&line=7", 200)
	get("GET", "/api/code/resolve?file=src/Repo/UserRepo.php", 400)
}

func TestMCPTools(t *testing.T) {
	m, _ := setup(t, Config{})
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	m.RegisterMCPTools(server)
	ct, st := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "c", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	tests := []struct {
		tool    string
		args    map[string]any
		want    string
		isError bool
	}{
		{"code_index_status", nil, `"state": "ready"`, false},
		{"code_search", map[string]any{"query": "persist"}, `App\\Repo\\UserRepo::persist`, false},
		{"code_symbol", map[string]any{"symbol": `App\Service\UserService`}, `"kind": "extends"`, false},
		{"code_callers", map[string]any{"symbol": `App\Repo\UserRepo::persist`}, `"confidence": "name"`, false},
		{"code_callees", map[string]any{"symbol": "example.com/app/cmd.main"}, `example.com/app/internal/repo.New`, false},
		{"code_file_outline", map[string]any{"path": "go/internal/repo/repo.go"}, `"fqn": "example.com/app/internal/repo.Repo"`, false},
		{"code_dependencies", map[string]any{"symbol": `App\Service\UserService`}, `ns:App\\Repo`, false},
		{"code_resolve_frame", map[string]any{"file": "src/Service/Base.php", "line": 9}, `App\\Service\\Base::boot`, false},
		{"code_symbol", map[string]any{"symbol": `No\Such`}, "not found", true},
	}
	for _, tc := range tests {
		t.Run(tc.tool, func(t *testing.T) {
			res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{Name: tc.tool, Arguments: tc.args})
			if err != nil {
				t.Fatal(err)
			}
			var sb strings.Builder
			for _, c := range res.Content {
				if tc, ok := c.(*sdkmcp.TextContent); ok {
					sb.WriteString(tc.Text)
				}
			}
			if res.IsError != tc.isError || !strings.Contains(sb.String(), tc.want) {
				t.Errorf("isError=%v, want %v; %q not in:\n%s", res.IsError, tc.isError, tc.want, sb.String())
			}
		})
	}
}

// A syntax error that sends tree-sitter's recovery into minutes (a dropped
// quote in sentry-php's Options.php, testdata/broken/; 2 min unbounded right
// after the intact file went through the same parser) stops at the parse
// timeout: the file is marked parse_error, keeps what the first pass found,
// and the rest of the root indexes.
func TestIndex_BrokenFileStopsAtTimeout(t *testing.T) {
	dir := copyTree(t)
	b, err := os.ReadFile("testdata/broken/Options.php")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src/Options.php"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	// The slow recovery shows when the same parser just parsed the intact
	// file, as in a run over a tree: A_Options.php sorts right before it.
	orig, err := os.ReadFile("testdata/broken/Options.orig.php")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src/A_Options.php"), orig, 0o644); err != nil {
		t.Fatal(err)
	}
	db := openDB(t)
	m := New(Config{Roots: []Root{{Name: "app", Path: dir}}, ParseTimeout: time.Second}) // the intact file takes ~20 ms, ~10x that with -race
	started := time.Now()
	if err := m.OnInit(db); err != nil {
		t.Fatal(err)
	}
	m.Indexer().Wait()
	if took := time.Since(started); took > 20*time.Second {
		t.Fatalf("indexing took %v", took)
	}
	st, _ := m.ix.Status("app")
	if st.State != StateReady || st.Files != 10 || st.ParseErrors != 1 {
		t.Fatalf("status %+v", st)
	}
	var perr, syms int
	_ = db.QueryRow(`SELECT parse_error, (SELECT COUNT(*) FROM code_symbols s WHERE s.file_id = f.id) FROM code_files f WHERE path = 'src/Options.php'`).Scan(&perr, &syms)
	if perr != 1 || syms == 0 {
		t.Errorf("broken file: parse_error=%d symbols=%d", perr, syms)
	}
	mustLookup(t, m, `App\Service\UserService::save`) // the rest is indexed
}

// A cancelled run stops a parse in progress at once.
func TestParsers_Cancel(t *testing.T) {
	b, err := os.ReadFile("testdata/broken/Options.php")
	if err != nil {
		t.Fatal(err)
	}
	orig, err := os.ReadFile("testdata/broken/Options.orig.php")
	if err != nil {
		t.Fatal(err)
	}
	ps := newParsers(time.Minute)
	if _, err := ps.parse("php", orig, ""); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer ps.watch(ctx)()
	time.AfterFunc(100*time.Millisecond, cancel)
	started := time.Now()
	res, err := ps.parse("php", b, "")
	if err != nil && !errors.Is(err, errParseStopped) {
		t.Fatal(err)
	}
	if took := time.Since(started); took > 10*time.Second || (err == nil && res.Stopped != "cancelled") {
		t.Fatalf("after %v: err=%v stopped=%q", took, err, res.Stopped)
	}
}

// Two runs within one second: the whole-graph cache must not keep old ids.
func TestSubgraph_CacheAcrossFastRuns(t *testing.T) {
	m, dir := setup(t, Config{})
	if _, err := m.graphs.Subgraph(GraphQuery{Root: "app", Level: LevelSymbol}); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "src/Repo/UserRepo.php")
	src, _ := os.ReadFile(f)
	if err := os.WriteFile(f, []byte(strings.Replace(string(src), "class UserRepo", "class UserRepo\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	m.ix.Reindex("app")
	m.ix.Wait()
	find := mustLookup(t, m, `App\Repo\UserRepo::find`)
	g, err := m.graphs.Subgraph(GraphQuery{Root: "app", Level: LevelSymbol})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range g.Nodes {
		if n.SymbolID == find.ID {
			return
		}
	}
	t.Errorf("node of the re-parsed UserRepo::find (id %d) missing: stale cache", find.ID)
}

func TestTop(t *testing.T) {
	m, _ := setup(t, Config{})
	top, err := m.graphs.Top("app", 50)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for i, s := range top {
		got[s.FQN] = s.Callers
		if s.Kind != KindFunction && s.Kind != KindMethod {
			t.Errorf("%s: kind %s", s.FQN, s.Kind)
		}
		if i > 0 && top[i-1].Callers < s.Callers {
			t.Errorf("not ordered: %v", top)
		}
	}
	// Calls from tests do not count; test symbols never appear.
	if got[`App\Repo\UserRepo::find`] != 1 || got["example.com/app/internal/repo.New"] != 1 {
		t.Errorf("counts: %v", got)
	}
	for _, name := range []string{`App\Tests\helperInTest`, `App\Tests\UserRepoTest::testFind`, "example.com/app/internal/repo.TestNew"} {
		if _, ok := got[name]; ok {
			t.Errorf("test symbol %s listed", name)
		}
	}
}

func TestSubgraph_Expand(t *testing.T) {
	m, _ := setup(t, Config{})
	byID := func(g *Graph) map[string]GraphNode {
		out := map[string]GraphNode{}
		for _, n := range g.Nodes {
			out[n.ID] = n
		}
		return out
	}
	g, err := m.graphs.Subgraph(GraphQuery{Root: "app"})
	if err != nil {
		t.Fatal(err)
	}
	nodes := byID(g)
	// App\Service: Saver, save, T, t, Kind, label, UserService + 4 methods, top, Base, boot, helper2.
	if n := nodes[`ns:App\Service`]; n.Size != 15 {
		t.Errorf("App\\Service size = %d, want 15", n.Size)
	}
	svc := mustLookup(t, m, `App\Service\UserService`)
	sid := "s:" + itoa(svc.ID)
	g, err = m.graphs.Subgraph(GraphQuery{Root: "app", Expand: map[string]bool{`ns:App\Service`: true, sid: true}})
	if err != nil {
		t.Fatal(err)
	}
	nodes = byID(g)
	if _, ok := nodes[`ns:App\Service`]; ok {
		t.Error("expanded namespace still a node")
	}
	if n := nodes["s:"+itoa(mustLookup(t, m, `App\Service\Base`).ID)]; n.Group != `ns:App\Service` || n.Size != 2 {
		t.Errorf("Base: %+v", n)
	}
	save := nodes["s:"+itoa(mustLookup(t, m, `App\Service\UserService::save`).ID)]
	if save.Group != sid {
		t.Errorf("save not expanded under its class: %+v", save)
	}
	if _, ok := nodes[sid]; ok {
		t.Error("expanded class still a node")
	}
	for _, n := range g.Nodes {
		if n.Group != "" && (n.Kind == KindNamespace || n.Kind == KindPackage) {
			t.Errorf("namespace row shown inside an expansion: %+v", n)
		}
	}
	for _, e := range g.Edges {
		if e.Kind == EdgeImports && (nodes[e.From].Group != "" || nodes[e.To].Group != "") {
			t.Errorf("expanded namespace kept its imports edge: %+v", e)
		}
	}
	// Go: an expanded package must not show its package row either.
	gg, err := m.graphs.Subgraph(GraphQuery{Root: "app", Expand: map[string]bool{"ns:example.com/app/internal/repo": true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range gg.Nodes {
		if n.Kind == KindPackage && n.Group != "" {
			t.Errorf("package row shown inside an expansion: %+v", n)
		}
	}
	// No level finer than symbol: the expand is ignored, the node stays.
	sg, err := m.graphs.Subgraph(GraphQuery{Root: "app", Level: LevelSymbol, Expand: map[string]bool{sid: true}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := byID(sg)[sid]; !ok {
		t.Error("expand at symbol level dropped the node")
	}
	if _, ok := nodes[`ns:App\Repo`]; !ok {
		t.Error("other namespaces must stay folded")
	}
	found := false
	for _, e := range g.Edges {
		if e.From == save.ID && e.To == `ns:App\Repo` && e.Kind == EdgeCalls {
			found = true
		}
	}
	if !found {
		t.Errorf("save -> ns:App\\Repo call edge missing: %+v", g.Edges)
	}
}
