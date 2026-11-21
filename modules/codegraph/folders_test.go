package codegraph

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/auth"
)

// baseTree makes base/{proj (a copy of testdata/app), other, .hidden,
// vendor, file.txt, inlink -> proj, outlink -> outside} and an outside
// folder next to base.
func baseTree(t *testing.T) (base, outside string) {
	t.Helper()
	top := t.TempDir()
	base = filepath.Join(top, "base")
	outside = filepath.Join(top, "outside")
	for _, d := range []string{base, outside, filepath.Join(base, "other"), filepath.Join(base, ".hidden"), filepath.Join(base, "vendor")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Rename(copyTree(t), filepath.Join(base, "proj")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.php"), []byte("<?php function secret() {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "proj"), filepath.Join(base, "inlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "outlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(base, "proj", "escape")); err != nil {
		t.Fatal(err)
	}
	base, _ = filepath.EvalSymlinks(base)
	outside, _ = filepath.EvalSymlinks(outside)
	return base, outside
}

type picker struct {
	t   *testing.T
	m   *Module
	mux *http.ServeMux
}

func newPicker(t *testing.T, cfg Config) *picker {
	t.Helper()
	if cfg.Roots == nil {
		cfg.Roots = []Root{}
	}
	m := New(cfg)
	if err := m.OnInit(openDB(t)); err != nil {
		t.Fatal(err)
	}
	return pickerFor(t, m)
}

func pickerFor(t *testing.T, m *Module) *picker {
	mux := http.NewServeMux()
	m.RegisterProtectedRoutes(mux, func(h http.Handler) http.Handler { return h })
	return &picker{t: t, m: m, mux: mux}
}

func (p *picker) do(method, url, body string, user *auth.User, hdr map[string]string) *httptest.ResponseRecorder {
	p.t.Helper()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	// From this machine unless the test says otherwise.
	req.RemoteAddr, req.Host = "127.0.0.1:50000", "localhost:8000"
	for k, v := range hdr {
		switch k {
		case "RemoteAddr":
			req.RemoteAddr = v
		case "Host":
			req.Host = v
		default:
			req.Header.Set(k, v)
		}
	}
	if user != nil {
		req = req.WithContext(auth.WithUser(req.Context(), user))
	}
	rec := httptest.NewRecorder()
	p.mux.ServeHTTP(rec, req)
	return rec
}

func (p *picker) want(rec *httptest.ResponseRecorder, status int, code string) {
	p.t.Helper()
	if rec.Code != status {
		p.t.Fatalf("status %d, want %d: %s", rec.Code, status, rec.Body)
	}
	if code != "" && !strings.Contains(rec.Body.String(), `"code":"`+code+`"`) {
		p.t.Fatalf("want code %s: %s", code, rec.Body)
	}
}

func TestInBase_TraversalAndSymlinks(t *testing.T) {
	base, _ := baseTree(t)
	for _, rel := range []string{"..", "../outside", "proj/../../outside", "/etc", "outlink", "proj/escape", "proj/escape/sub"} {
		if p, err := inBase(base, rel); err == nil {
			t.Errorf("inBase(%q) = %q, want refused", rel, p)
		}
	}
	for rel, want := range map[string]string{"": base, "proj": filepath.Join(base, "proj"), "inlink": filepath.Join(base, "proj"), "proj/..": base, "./other": filepath.Join(base, "other"),
		// Cleaned before symlinks are followed: escape/.. is proj, not outside's parent.
		"proj/escape/..": filepath.Join(base, "proj")} {
		if p, err := inBase(base, rel); err != nil || p != want {
			t.Errorf("inBase(%q) = %q, %v, want %q", rel, p, err, want)
		}
	}
	if _, err := inBase(base, "file.txt"); err != errNotDir {
		t.Errorf("file: %v", err)
	}
	if _, err := inBase("", "proj"); err == nil {
		t.Error("no base: allowed")
	}
}

func TestFolders_Browse(t *testing.T) {
	base, _ := baseTree(t)
	p := newPicker(t, Config{BaseDir: base})
	rec := p.do("GET", "/api/code/folders", "", nil, nil)
	p.want(rec, 200, "")
	var res FoldersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	got := map[string]Folder{}
	for _, f := range res.Folders {
		got[f.Name] = f
	}
	if res.BaseDir != base || res.Path != "" || res.Parent != nil {
		t.Errorf("head: %+v", res)
	}
	if _, ok := got["outlink"]; ok {
		t.Error("symlink to outside listed")
	}
	if _, ok := got["file.txt"]; ok {
		t.Error("file listed")
	}
	if !got[".hidden"].Hidden || !got["vendor"].Excluded || got["proj"].Hidden || got["proj"].Excluded {
		t.Errorf("marks: %+v", got)
	}
	if got["inlink"].Path != "proj" {
		t.Errorf("inside symlink path: %+v", got["inlink"])
	}
	rec = p.do("GET", "/api/code/folders?path=proj", "", nil, nil)
	p.want(rec, 200, "")
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Parent == nil || *res.Parent != "" || res.Path != "proj" {
		t.Errorf("proj: %+v", res)
	}
	for _, f := range res.Folders {
		if f.Name == "escape" {
			t.Error("nested symlink to outside listed")
		}
	}
	p.want(p.do("GET", "/api/code/folders?path=../outside", "", nil, nil), 400, "outside_base")
	p.want(p.do("GET", "/api/code/folders?path=outlink", "", nil, nil), 400, "outside_base")
	p.want(p.do("GET", "/api/code/folders?path=nope", "", nil, nil), 404, "no_such_folder")
}

func TestFolders_AddRemoveRoot(t *testing.T) {
	base, outside := baseTree(t)
	db := openDB(t)
	cfgRoot := filepath.Join(base, "other")
	m := New(Config{BaseDir: base, Roots: []Root{{Name: "cfg", Path: cfgRoot}}})
	if err := m.OnInit(db); err != nil {
		t.Fatal(err)
	}
	p := pickerFor(t, m)

	for _, body := range []string{`{"path":"../outside"}`, `{"path":"outlink"}`, `{"path":"proj/escape"}`, `{"path":"` + outside + `"}`} {
		p.want(p.do("POST", "/api/code/roots", body, nil, nil), 400, "outside_base")
	}
	p.want(p.do("POST", "/api/code/roots", `{"path":"other"}`, nil, nil), 409, "root_exists")
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj","name":"a/b"}`, nil, nil), 400, "bad_name")
	p.want(p.do("POST", "/api/code/roots", `{"path":""}`, nil, nil), 400, "base_root")
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj/.."}`, nil, nil), 400, "base_root")

	rec := p.do("POST", "/api/code/roots", `{"path":"inlink","name":"proj"}`, nil, nil)
	p.want(rec, 201, "")
	var st Status
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	if st.Name != "proj" || st.Path != filepath.Join(base, "proj") || st.Source != SourceUI {
		t.Errorf("added: %+v", st)
	}
	m.Indexer().Wait()
	if st, _ := m.Indexer().Status("proj"); st.State != StateReady || st.Symbols == 0 {
		t.Errorf("indexed: %+v", st)
	}
	// The symlink to outside inside the root is not followed.
	if _, err := m.graphs.Lookup("secret", "proj"); err == nil {
		t.Error("indexed a file outside base through a symlink")
	}
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj","name":"again"}`, nil, nil), 409, "root_exists")
	p.want(p.do("POST", "/api/code/roots/proj/reindex", "", nil, nil), 202, "")
	m.Indexer().Wait()

	var roots RootsResponse
	_ = json.Unmarshal(p.do("GET", "/api/code/roots", "", nil, nil).Body.Bytes(), &roots)
	if len(roots.Roots) != 2 || roots.Roots[0].Source != SourceConfig || roots.Roots[1].Source != SourceUI || !roots.Picker.Enabled || !roots.Picker.CanManage {
		t.Errorf("roots: %+v", roots)
	}
	p.want(p.do("DELETE", "/api/code/roots/cfg", "", nil, nil), 403, "config_root")

	// Survives a restart.
	m2 := New(Config{BaseDir: base, Roots: []Root{{Name: "cfg", Path: cfgRoot}}})
	if err := m2.OnInit(db); err != nil {
		t.Fatal(err)
	}
	m2.Indexer().Wait()
	if st, ok := m2.Indexer().Status("proj"); !ok || st.Source != SourceUI || st.Symbols == 0 {
		t.Errorf("after restart: %+v %v", st, ok)
	}
	// Outside a new base_dir it is skipped, not dropped from the table.
	m3 := New(Config{BaseDir: filepath.Join(base, "other"), Roots: []Root{}})
	if err := m3.OnInit(db); err != nil {
		t.Fatal(err)
	}
	if _, ok := m3.Indexer().Status("proj"); ok {
		t.Error("UI root outside the new base_dir loaded")
	}
	m3.Indexer().Wait()

	p2 := pickerFor(t, m2)
	p2.want(p2.do("DELETE", "/api/code/roots/proj", "", nil, nil), 204, "")
	m2.Indexer().Wait()
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM code_files WHERE root = 'proj'`).Scan(&n)
	if n != 0 {
		t.Errorf("files left after remove: %d", n)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM code_ui_roots`).Scan(&n)
	if n != 0 {
		t.Errorf("ui roots left: %d", n)
	}
	p2.want(p2.do("DELETE", "/api/code/roots/proj", "", nil, nil), 404, "unknown_root")
	// Added again under the same name: fresh data, no stale symbol cache.
	p2.want(p2.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, nil), 201, "")
	m2.Indexer().Wait()
	if _, err := m2.graphs.Subgraph(GraphQuery{Root: "proj"}); err != nil {
		t.Error(err)
	}
}

func TestFolders_RemoveWhileIndexing(t *testing.T) {
	base, _ := baseTree(t)
	p := newPicker(t, Config{BaseDir: base})
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, nil), 201, "")
	p.want(p.do("DELETE", "/api/code/roots/proj", "", nil, nil), 204, "")
	p.m.Indexer().Wait()
	var n int
	_ = p.m.ix.db.QueryRow(`SELECT COUNT(*) FROM code_files WHERE root = 'proj'`).Scan(&n)
	if n != 0 {
		t.Errorf("files left: %d", n)
	}
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, nil), 201, "")
	p.m.Indexer().Wait()
}

func TestFolders_Guards(t *testing.T) {
	base, _ := baseTree(t)
	// Picker off: 404 with the hint, roots still answer.
	off := newPicker(t, Config{Roots: []Root{{Name: "o", Path: filepath.Join(base, "other")}}})
	off.want(off.do("GET", "/api/code/folders", "", nil, nil), 404, "picker_off")
	off.want(off.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, nil), 404, "picker_off")
	off.want(off.do("DELETE", "/api/code/roots/o", "", nil, nil), 404, "picker_off")
	var roots RootsResponse
	_ = json.Unmarshal(off.do("GET", "/api/code/roots", "", nil, nil).Body.Bytes(), &roots)
	if roots.Picker.Enabled || roots.Picker.Hint == "" {
		t.Errorf("picker off: %+v", roots.Picker)
	}
	if New(Config{BaseDir: filepath.Join(base, "file.txt"), Roots: []Root{}}).Usable() {
		t.Error("a file as base_dir is usable")
	}

	// Another origin.
	p := newPicker(t, Config{BaseDir: base})
	p.want(p.do("GET", "/api/code/folders", "", nil, map[string]string{"Sec-Fetch-Site": "cross-site"}), 403, "cross_origin")
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, map[string]string{"Origin": "http://evil.test"}), 403, "cross_origin")

	// Without auth: only from this machine, whatever the listen address.
	lan := map[string]string{"RemoteAddr": "192.168.1.30:40000", "Host": "192.168.1.20:8000"}
	p.want(p.do("GET", "/api/code/folders", "", nil, lan), 403, "not_local")
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, lan), 403, "not_local")
	p.want(p.do("DELETE", "/api/code/roots/x", "", nil, lan), 403, "not_local")
	// DNS rebinding: loopback peer, foreign Host, the browser calls it same-origin.
	rebind := map[string]string{"Host": "rebind.evil.test:8000", "Origin": "http://rebind.evil.test:8000", "Sec-Fetch-Site": "same-origin"}
	p.want(p.do("GET", "/api/code/folders", "", nil, rebind), 403, "not_local")
	p.want(p.do("POST", "/api/code/roots", `{"path":"proj"}`, nil, rebind), 403, "not_local")
	p.want(p.do("GET", "/api/code/folders", "", nil, map[string]string{"RemoteAddr": "[::1]:5000", "Host": "[::1]:8000"}), 200, "")
	_ = json.Unmarshal(p.do("GET", "/api/code/roots", "", nil, lan).Body.Bytes(), &roots)
	if !roots.Picker.Enabled || roots.Picker.CanManage || roots.Picker.BaseDir != "" || roots.Picker.Hint != LocalOnlyHint {
		t.Errorf("LAN picker: %+v", roots.Picker)
	}

	// Auth on: admins only.
	ap := newPicker(t, Config{BaseDir: base, AuthOn: true, Admins: []string{"Ann", "bob@x.io"}})
	ann := &auth.User{Username: "ann"}
	bob := &auth.User{Username: "b", Email: "BOB@x.io"}
	eve := &auth.User{Username: "eve", Email: "eve@x.io"}
	ap.want(ap.do("GET", "/api/code/folders", "", eve, nil), 403, "not_admin")
	ap.want(ap.do("POST", "/api/code/roots", `{"path":"proj"}`, eve, nil), 403, "not_admin")
	ap.want(ap.do("GET", "/api/code/folders", "", ann, nil), 200, "")
	ap.want(ap.do("GET", "/api/code/folders", "", ann, lan), 200, "") // with auth, an admin from another machine
	ap.want(ap.do("POST", "/api/code/roots", `{"path":"proj"}`, bob, nil), 201, "")
	ap.want(ap.do("DELETE", "/api/code/roots/proj", "", eve, nil), 403, "not_admin")
	_ = json.Unmarshal(ap.do("GET", "/api/code/roots", "", eve, nil).Body.Bytes(), &roots)
	if !roots.Picker.Enabled || roots.Picker.CanManage {
		t.Errorf("eve picker: %+v", roots.Picker)
	}
	ap.m.Indexer().Wait()
	none := newPicker(t, Config{BaseDir: base, AuthOn: true})
	none.want(none.do("GET", "/api/code/folders", "", ann, nil), 403, "not_admin")
}
