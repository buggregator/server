package xdebugtrace

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/buggregator/go-buggregator/internal/event"
	"github.com/buggregator/go-buggregator/internal/storage"
)

type testEnv struct {
	dir   string
	mod   *Module
	store *storage.SQLiteStore
	mux   *http.ServeMux
	ts    float64
}

func newEnv(t *testing.T, limits Limits) *testEnv {
	t.Helper()
	base := t.TempDir()
	db, err := storage.Open(filepath.Join(base, "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE events (
		uuid TEXT PRIMARY KEY, type TEXT NOT NULL, payload TEXT NOT NULL,
		timestamp TEXT NOT NULL, project TEXT, is_pinned INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatal(err)
	}
	e := &testEnv{dir: filepath.Join(base, "traces"), store: storage.NewSQLiteStore(db), mux: http.NewServeMux()}
	e.mod = New(e.dir, limits)
	if err := e.mod.OnInit(db); err != nil {
		t.Fatal(err)
	}
	e.mod.RegisterRoutes(e.mux, e.store)
	e.mod.RegisterProtectedRoutes(e.mux, func(h http.Handler) http.Handler { return h })
	e.store.OnDelete(e.mod.OnEventsDeleted) // as main.go wires it
	return e
}

func (e *testEnv) upload(t *testing.T, body []byte, hdr map[string]string) (*event.Incoming, error) {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	r.Header.Set(headerDetectedType, Type)
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	h := e.mod.HTTPHandler()
	if !h.Match(r) {
		t.Fatal("handler does not match")
	}
	in, err := h.Handle(r)
	if err != nil {
		return nil, err
	}
	e.ts++
	ev := event.Event{UUID: in.UUID, Type: in.Type, Payload: in.Payload, Timestamp: e.ts}
	if err := e.store.Store(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	e.mod.OnEventStored(ev)
	return in, nil
}

func (e *testEnv) files(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(e.dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, x := range entries {
		out = append(out, x.Name())
	}
	sort.Strings(out)
	return out
}

func (e *testEnv) get(t *testing.T, path string, out any) int {
	t.Helper()
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if out != nil && w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
			t.Fatalf("%s: %v: %s", path, err, w.Body.String())
		}
	}
	return w.Code
}

func TestModule_UploadAndAPI(t *testing.T) {
	e := newEnv(t, Limits{})
	body := gz(t, sample)
	in, err := e.upload(t, body, map[string]string{"X-Buggregator-Trace-Url": "GET /users"})
	if err != nil {
		t.Fatal(err)
	}

	var p Payload
	json.Unmarshal(in.Payload, &p)
	if p.File != in.UUID+extGzip || !p.Compressed || p.Size != int64(len(body)) || p.URL != "GET /users" || p.Summary == nil || p.Summary.Calls != 6 || p.Summary.Index != nil {
		t.Fatalf("payload = %+v", p)
	}
	stored, err := os.ReadFile(filepath.Join(e.dir, p.File))
	if err != nil || !bytes.Equal(stored, body) {
		t.Fatalf("stored file differs from the upload (err %v)", err)
	}
	if got := e.files(t); len(got) != 2 || got[0] != in.UUID+extIndex || got[1] != p.File {
		t.Errorf("files = %v, want the trace and its index", got)
	}

	prev, _ := previewMapper{}.ToPreview(in.Payload)
	if !strings.Contains(string(prev), `"calls":6`) || !strings.Contains(string(prev), `"url":"GET /users"`) {
		t.Errorf("preview = %s", prev)
	}

	base := "/api/xdebug-traces/" + in.UUID
	var sum struct {
		Summary Summary `json:"summary"`
	}
	if c := e.get(t, base+"/summary", &sum); c != 200 || sum.Summary.Main != "/app/index.php" {
		t.Errorf("summary: %d %+v", c, sum)
	}
	var top Page
	if c := e.get(t, base+"/tree", &top); c != 200 || len(top.Nodes) != 1 || top.Nodes[0].Func != "{main}" {
		t.Fatalf("tree: %d %+v", c, top)
	}
	var main Page
	if c := e.get(t, fmt.Sprintf("%s/tree?parent=%d", base, top.Nodes[0].Offset), &main); c != 200 || len(main.Nodes) != 1 || main.Nodes[0].Children != 3 {
		t.Errorf("tree of {main}: %d %+v", c, main)
	}
	var found Page
	if c := e.get(t, base+"/search?q=pdo", &found); c != 200 || len(found.Nodes) != 1 || found.Nodes[0].Func != "PDO->query" {
		t.Errorf("search: %d %+v", c, found)
	}

	for path, want := range map[string]int{
		base + "/tree?parent=1":                    400, // not an entry line
		base + "/tree?parent=99999999":             400, // past the end
		base + "/tree?limit=x":                     400,
		base + "/search":                           400,
		"/api/xdebug-traces/no-such/summary":       404,
		"/api/xdebug-traces/..%2Fdb.sqlite/tree":   404,
		"/api/xdebug-traces/" + in.UUID + "x/tree": 404,
	} {
		if c := e.get(t, path, nil); c != want {
			t.Errorf("%s: %d, want %d", path, c, want)
		}
	}

	// Deleting the event removes its file.
	if err := e.store.Delete(context.Background(), in.UUID); err != nil {
		t.Fatal(err)
	}
	if got := e.files(t); len(got) != 0 {
		t.Errorf("files after delete = %v", got)
	}
}

// bigTrace has {main} with n direct calls, enough for index points.
func bigTrace(n int) string {
	var b strings.Builder
	b.WriteString("Version: 3.4.5\nFile format: 4\nTRACE START [2026-11-08 10:00:00.000001]\n")
	b.WriteString("1\t0\t0\t0.000100\t400000\t{main}\t1\t\t/app/big.php\t0\t0\n")
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "2\t%d\t0\t0.%06d\t400000\tf%d\t1\t\t/app/big.php\t%d\t0\n", i, i+100, i, i)
		fmt.Fprintf(&b, "2\t%d\t1\t0.%06d\t400000\n", i, i+101)
	}
	fmt.Fprintf(&b, "1\t0\t1\t0.%06d\t400000\n\t\t\t0.%06d\t400000\nTRACE END   [2026-11-08 10:00:01.000000]\n", n+200, n+201)
	return b.String()
}

func TestModule_IndexAndPaging(t *testing.T) {
	e := newEnv(t, Limits{})
	in, err := e.upload(t, []byte(bigTrace(2500)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := e.files(t); len(got) != 2 || got[0] != in.UUID+extIndex || got[1] != in.UUID+extPlain {
		t.Fatalf("files = %v", got)
	}
	base := "/api/xdebug-traces/" + in.UUID
	var sum struct {
		Compressed bool    `json:"compressed"`
		Summary    Summary `json:"summary"`
	}
	if c := e.get(t, base+"/summary", &sum); c != 200 || sum.Compressed || len(sum.Summary.Index) != 3 {
		t.Fatalf("summary: %d index %d", c, len(sum.Summary.Index))
	}
	var top Page
	e.get(t, base+"/tree", &top)
	parent := fmt.Sprintf("%s/tree?parent=%d&limit=1000", base, top.Nodes[0].Offset)
	seen := 0
	var page Page
	if c := e.get(t, parent, &page); c != 200 {
		t.Fatalf("tree: %d", c)
	}
	for {
		seen += len(page.Nodes)
		if page.Next == nil {
			break
		}
		next := page.Next
		page = Page{}
		if c := e.get(t, fmt.Sprintf("%s/tree?from=%d&limit=1000", base, *next), &page); c != 200 {
			t.Fatalf("next page: %d", c)
		}
	}
	if seen != 2500 {
		t.Errorf("paged %d callees, want 2500", seen)
	}
}

func TestModule_RejectsAndLeavesNoFiles(t *testing.T) {
	e := newEnv(t, Limits{MaxUpload: 1024})
	if _, err := e.upload(t, []byte("TRACE START [x]\n0.0001 400000 -> {main}() /app/index.php:0\n"), nil); !errors.Is(err, ErrNotFormat1) {
		t.Errorf("format 0: err = %v, want ErrNotFormat1", err)
	}
	if _, err := e.upload(t, []byte(bigTrace(100)), nil); err == nil || !strings.Contains(err.Error(), "max_upload") {
		t.Errorf("too large: err = %v", err)
	}
	if got := e.files(t); len(got) != 0 {
		t.Errorf("files left = %v", got)
	}
}

func TestModule_MaxSizeCutsTheSummary(t *testing.T) {
	e := newEnv(t, Limits{MaxSize: 4096})
	in, err := e.upload(t, gz(t, bigTrace(500)), nil)
	if err != nil {
		t.Fatal(err)
	}
	var p Payload
	json.Unmarshal(in.Payload, &p)
	if !p.Summary.Truncated || p.Summary.Calls >= 500 {
		t.Errorf("summary = %+v", p.Summary)
	}
	// The file is kept whole; the tree stops at the same limit.
	var top Page
	if c := e.get(t, "/api/xdebug-traces/"+in.UUID+"/tree", &top); c != 200 || !top.Truncated {
		t.Errorf("tree: %d truncated=%v", c, top.Truncated)
	}
}

func TestModule_KeepsTheNewest(t *testing.T) {
	e := newEnv(t, Limits{Keep: 2})
	var uuids []string
	for i := 0; i < 4; i++ {
		in, err := e.upload(t, []byte(sample), nil)
		if err != nil {
			t.Fatal(err)
		}
		uuids = append(uuids, in.UUID)
		if i == 0 {
			e.store.Pin(context.Background(), in.UUID)
		}
	}
	// Pinned uuids[0] stays; uuids[1] is the oldest unpinned past the limit.
	left, _ := e.store.FindAll(context.Background(), event.FindOptions{Type: Type})
	got := map[string]bool{}
	for _, ev := range left {
		got[ev.UUID] = true
	}
	if len(got) != 3 || !got[uuids[0]] || got[uuids[1]] || !got[uuids[2]] || !got[uuids[3]] {
		t.Errorf("events left = %v (uploaded %v)", got, uuids)
	}
	if files := e.files(t); len(files) != 6 { // trace + index each
		t.Errorf("files = %v", files)
	}
}

func TestModule_SweepRemovesOrphans(t *testing.T) {
	e := newEnv(t, Limits{})
	in, err := e.upload(t, []byte(sample), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"0badf00d-0000-4000-8000-000000000000.xt.gz", "0badf00d-0000-4000-8000-000000000001.part"} {
		os.WriteFile(filepath.Join(e.dir, name), []byte("x"), 0o644)
	}
	e.mod.sweep()
	if got := e.files(t); len(got) != 2 || got[1] != in.UUID+extPlain {
		t.Errorf("files after sweep = %v", got)
	}
}
