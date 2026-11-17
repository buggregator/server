package selfupdate

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/buggregator/server/v2/internal/storage"
	"github.com/buggregator/server/v2/internal/updates"
)

func TestDecide(t *testing.T) {
	base := Facts{Version: "1.0.0", Release: true, CheckOn: true, Setting: SettingAuto, HTTPAddr: "127.0.0.1:8000", GOOS: "linux", Exe: "/opt/bg/buggregator", Writable: true}
	tests := []struct {
		name    string
		change  func(*Facts)
		mode    string
		install string
		reason  string
	}{
		{"loopback binary", func(*Facts) {}, ModeSelf, InstallBinary, ""},
		{"check off", func(f *Facts) { f.CheckOn = false }, ModeOff, InstallBinary, "UPDATE_CHECK_ENABLED"},
		{"dev build", func(f *Facts) { f.Version, f.Release = "dev", false }, ModeOff, InstallBinary, "no release version"},
		{"setting off", func(f *Facts) { f.Setting = SettingOff }, ModeOff, InstallBinary, "UPDATE_SELF"},
		{"docker", func(f *Facts) { f.Container = InstallDocker }, ModeManual, InstallDocker, "container"},
		{"docker even with on", func(f *Facts) { f.Container, f.Setting = InstallDocker, SettingOn }, ModeManual, InstallDocker, "container"},
		{"kubernetes", func(f *Facts) { f.Container = InstallKubernetes }, ModeManual, InstallKubernetes, "Helm"},
		{"read-only dir", func(f *Facts) { f.Writable = false }, ModeManual, InstallBinary, "/opt/bg"},
		{"all interfaces, no auth", func(f *Facts) { f.HTTPAddr = ":8000" }, ModeManual, InstallBinary, "update.self_update: on (env UPDATE_SELF=on)"},
		{"all interfaces, auth", func(f *Facts) { f.HTTPAddr, f.AuthOn = ":8000", true }, ModeSelf, InstallBinary, ""},
		{"all interfaces, on", func(f *Facts) { f.HTTPAddr, f.Setting = "0.0.0.0:8000", SettingOn }, ModeSelf, InstallBinary, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := base
			tt.change(&f)
			m := Decide(f)
			if m.Mode != tt.mode || m.Install != tt.install || !strings.Contains(m.Reason, tt.reason) {
				t.Errorf("Decide = %+v, want mode %s install %s reason ~%q", m, tt.mode, tt.install, tt.reason)
			}
		})
	}
	f := base
	f.GOOS = "windows"
	if m := Decide(f); m.Mode != ModeSelf || !m.RestartManual {
		t.Errorf("windows: %+v, want self with restart_manual", m)
	}
}

func TestLoopbackOnly(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:8000": true, "localhost:8000": true, "[::1]:8000": true,
		":8000": false, "0.0.0.0:8000": false, "192.168.1.5:8000": false, "[::]:8000": false,
	} {
		if got := LoopbackOnly(addr); got != want {
			t.Errorf("LoopbackOnly(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestCommands(t *testing.T) {
	if c := Commands(InstallKubernetes, "linux", "/x", "2.1.0"); len(c) != 2 ||
		c[0].Command != "helm upgrade buggregator oci://ghcr.io/buggregator/helm-chart/buggregator --reset-then-reuse-values --set image.tag=2.1.0" ||
		c[1].Command != "helm upgrade buggregator oci://ghcr.io/buggregator/helm-chart/buggregator --reuse-values --set image.tag=2.1.0" ||
		c[0].Label != "Helm 3.14+" {
		t.Errorf("helm: %+v", c)
	}
	if c := Commands(InstallBinary, "linux", "/opt/bg/buggregator", "2.1.0"); !strings.Contains(c[0].Command, "BUGGREGATOR_INSTALL_DIR='/opt/bg' sh") {
		t.Errorf("shell: %q", c[0].Command)
	}
	if c := Commands(InstallBinary, "linux", "/opt/it's/buggregator", ""); !strings.Contains(c[0].Command, `BUGGREGATOR_INSTALL_DIR='/opt/it'\''s' sh`) {
		t.Errorf("shell quote: %q", c[0].Command)
	}
	if c := Commands(InstallBinary, "windows", "C:/it's/buggregator.exe", ""); !strings.Contains(c[0].Command, "'C:/it''s'") {
		t.Errorf("powershell quote: %q", c[0].Command)
	}
}

// release is a fake GitHub: releases/latest, the asset, checksums.txt and
// its signature.
type release struct {
	bin, sums, sig []byte
	noSig          bool
}

func (rel *release) serve(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			assets := []map[string]string{
				{"name": AssetName("linux", "amd64"), "browser_download_url": srv.URL + "/bin"},
				{"name": "checksums.txt", "browser_download_url": srv.URL + "/sums"},
			}
			if !rel.noSig {
				assets = append(assets, map[string]string{"name": "checksums.txt.sig", "browser_download_url": srv.URL + "/sig"})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": "v1.1.0", "html_url": "https://example/rel", "body": "notes", "assets": assets})
		case "/bin":
			w.Write(rel.bin)
		case "/sums":
			w.Write(rel.sums)
		case "/sig":
			w.Write(rel.sig)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newRelease builds a good release of a fake binary printing printed.
func newRelease(t *testing.T, priv ed25519.PrivateKey, printed string) *release {
	bin := []byte("#!/bin/sh\necho " + printed + "\n")
	sum := sha256.Sum256(bin)
	sums := []byte(hex.EncodeToString(sum[:]) + "  " + AssetName("linux", "amd64") + "\n" + strings.Repeat("0", 64) + "  other\n")
	return &release{bin: bin, sums: sums, sig: ed25519.Sign(priv, sums)}
}

type harness struct {
	u        *Updater
	exe      string
	mu       sync.Mutex
	phases   []string
	restarts int
}

func (h *harness) seen() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.phases...)
}

func setup(t *testing.T, rel *release, pub ed25519.PublicKey, dsn string, db *sql.DB) *harness {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake binary is a shell script")
	}
	t.Cleanup(func() { restarting.Store(false) })
	dir := t.TempDir()
	exe := filepath.Join(dir, "buggregator")
	if err := os.WriteFile(exe, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	bootAt(exe, "1.0.0", io.Discard)
	srv := rel.serve(t)
	c := updates.New("1.0.0", 0)
	c.SetURL(srv.URL + "/latest")
	if err := c.CheckNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := &harness{exe: exe}
	cfg := Config{Checker: c, DSN: dsn, DB: db,
		Broadcast: func(s State) { h.mu.Lock(); h.phases = append(h.phases, s.Phase); h.mu.Unlock() },
		Restart:   func() { h.mu.Lock(); h.restarts++; h.mu.Unlock() },
	}
	h.u = newUpdater(cfg, Mode{Mode: ModeSelf, Install: InstallBinary}, exe, "linux", "amd64")
	h.u.key = pub
	return h
}

func (h *harness) wait(t *testing.T) State {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := h.u.Info().State
		switch st.Phase {
		case PhaseFailed, PhaseRestarting, PhaseRestartRequired:
			return st
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("update did not finish")
	return State{}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUpdate_InstallsAndRestarts(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	rel := newRelease(t, priv, "1.1.0")

	dbPath := filepath.Join(t.TempDir(), "events.db")
	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE marker (v TEXT); INSERT INTO marker VALUES ('before')`); err != nil {
		t.Fatal(err)
	}

	h := setup(t, rel, pub, dbPath, db)
	if err := h.u.Start("1.1.0"); err != nil {
		t.Fatal(err)
	}
	if st := h.wait(t); st.Phase != PhaseRestarting || st.Version != "1.1.0" || st.Previous != "1.0.0" {
		t.Fatalf("state = %+v", st)
	}
	if got := read(t, h.exe); got != string(rel.bin) {
		t.Errorf("exe = %q, want the new binary", got)
	}
	if got := read(t, oldPath(h.exe)); got != "old" {
		t.Errorf("exe.old = %q, want the old binary", got)
	}
	if _, err := os.Stat(dbPath + ".pre-1.0.0.bak"); err != nil {
		t.Errorf("database copy: %v", err)
	}
	r, ok := readRecord(h.exe)
	if !ok || r.Phase != PhaseRestarting || r.DBBackup != dbPath+".pre-1.0.0.bak" {
		t.Errorf("state file = %+v", r)
	}
	h.mu.Lock()
	restarts := h.restarts
	h.mu.Unlock()
	if !Restarting() || restarts != 1 {
		t.Errorf("restart asked %d times, Restarting() = %v", restarts, Restarting())
	}
	want := []string{PhaseDownloading, PhaseVerifying, PhaseInstalling, PhaseRestarting}
	var got []string
	for _, p := range h.seen() {
		if len(got) == 0 || got[len(got)-1] != p {
			got = append(got, p)
		}
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("pushed phases %v, want %v", got, want)
	}
	if err := h.u.Start("1.1.0"); err != ErrBusy {
		t.Errorf("second start: %v, want ErrBusy", err)
	}
}

func TestUpdate_Refused(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	_, other, _ := ed25519.GenerateKey(nil)
	tests := []struct {
		name  string
		build func() *release
		step  string
		err   string
	}{
		{"bad signature", func() *release {
			r := newRelease(t, priv, "1.1.0")
			r.sig = ed25519.Sign(other, r.sums)
			return r
		}, PhaseVerifying, "signature"},
		{"no signature", func() *release {
			r := newRelease(t, priv, "1.1.0")
			r.noSig = true
			return r
		}, PhaseVerifying, "not signed"},
		{"wrong checksum", func() *release {
			r := newRelease(t, priv, "1.1.0")
			r.bin = append(r.bin, '#')
			return r
		}, PhaseVerifying, "sha256"},
		{"probe reports another version", func() *release { return newRelease(t, priv, "9.9.9") }, PhaseVerifying, `"9.9.9"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setup(t, tt.build(), pub, ":memory:", nil)
			if err := h.u.Start("1.1.0"); err != nil {
				t.Fatal(err)
			}
			st := h.wait(t)
			if st.Phase != PhaseFailed || st.FailedStep != tt.step || !strings.Contains(st.Error, tt.err) {
				t.Fatalf("state = %+v, want failed at %s with %q", st, tt.step, tt.err)
			}
			if got := read(t, h.exe); got != "old" {
				t.Errorf("exe changed: %q", got)
			}
			entries, _ := os.ReadDir(filepath.Dir(h.exe))
			if len(entries) != 1 {
				t.Errorf("left files: %v", entries)
			}
			if Restarting() {
				t.Error("restart asked after a failure")
			}
			// A failure frees the updater for another try.
			if err := h.u.Start("1.1.0"); err != nil {
				t.Errorf("retry: %v", err)
			}
			h.wait(t)
		})
	}
}

func TestStart_Rules(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	h := setup(t, newRelease(t, priv, "1.1.0"), pub, ":memory:", nil)
	if err := h.u.Start("1.0.5"); err != ErrNotLatest {
		t.Errorf("older version: %v, want ErrNotLatest", err)
	}
	h.u.mode = Mode{Mode: ModeManual, Reason: "why not"}
	if err := h.u.Start("1.1.0"); err == nil || !strings.Contains(err.Error(), "why not") {
		t.Errorf("manual mode: %v", err)
	}
}

// stage writes what a swap leaves: the new binary, the old one, a database
// with its copy and the restarting state.
func stage(t *testing.T, phase string) (exe, db string) {
	t.Helper()
	dir := t.TempDir()
	exe, db = filepath.Join(dir, "buggregator"), filepath.Join(dir, "events.db")
	for path, body := range map[string]string{exe: "new", oldPath(exe): "old", db: "migrated", db + ".bak": "before"} {
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	r := record{State: State{Phase: phase, Version: "1.1.0", Previous: "1.0.0"}, DB: db, DBBackup: db + ".bak"}
	if err := writeRecord(exe, r); err != nil {
		t.Fatal(err)
	}
	return exe, db
}

func swapExec(t *testing.T) *[]string {
	t.Helper()
	var calls []string
	prev := execFn
	execFn = func(exe string) error { calls = append(calls, exe); return nil }
	t.Cleanup(func() { execFn = prev })
	return &calls
}

func TestBoot_FailedStartRollsBack(t *testing.T) {
	calls := swapExec(t)
	exe, db := stage(t, PhaseRestarting)

	out := bootAt(exe, "1.1.0", io.Discard)
	if r, _ := readRecord(exe); r.Phase != phaseStarting {
		t.Fatalf("boot wrote %q, want starting", r.Phase)
	}
	fmt.Fprintln(out, "level=ERROR msg=\"failed to start HTTP server\"")
	rollbackPending() // what Exit(1) does before os.Exit

	if got := read(t, exe); got != "old" {
		t.Errorf("exe = %q, want the old binary back", got)
	}
	if got := read(t, db); got != "before" {
		t.Errorf("db = %q, want the copy back", got)
	}
	r, _ := readRecord(exe)
	if r.Phase != PhaseRolledBack || r.FailedStep != "starting" || !strings.Contains(r.Log, "failed to start HTTP server") {
		t.Errorf("state = %+v", r.State)
	}
	if len(*calls) != 1 || (*calls)[0] != exe {
		t.Errorf("exec calls %v, want the old binary run", *calls)
	}

	// The old binary starts and reports rolled_back.
	bootAt(exe, "1.0.0", io.Discard)
	if st := bootState(); st.Phase != PhaseRolledBack || st.Version != "1.1.0" {
		t.Errorf("old binary state = %+v", st)
	}
}

func TestBoot_DiedBeforeReadyRollsBackOnNextStart(t *testing.T) {
	calls := swapExec(t)
	exe, _ := stage(t, phaseStarting)
	bootAt(exe, "1.1.0", io.Discard)
	if got := read(t, exe); got != "old" || len(*calls) != 1 {
		t.Errorf("exe = %q, exec calls %v: want a rollback", got, *calls)
	}
}

func TestBoot_ReadyMarksDone(t *testing.T) {
	calls := swapExec(t)
	exe, db := stage(t, PhaseRestarting)
	bootAt(exe, "1.1.0", io.Discard)
	u := newUpdater(Config{}, Mode{Mode: ModeSelf}, exe, "linux", "amd64") // made before Ready, as in app.Run
	Ready()
	rollbackPending() // a later Exit no longer rolls back

	r, _ := readRecord(exe)
	if r.Phase != PhaseDone || r.Version != "1.1.0" || r.Previous != "1.0.0" {
		t.Errorf("state = %+v", r.State)
	}
	if _, err := os.Stat(db + ".bak"); !os.IsNotExist(err) {
		t.Errorf("database copy kept: %v", err)
	}
	if got := read(t, exe); got != "new" || len(*calls) != 0 {
		t.Errorf("exe = %q, exec %v: want no rollback", got, *calls)
	}
	if st := u.Info().State; st.Phase != PhaseDone || st.Previous != "1.0.0" {
		t.Errorf("API state = %+v", st)
	}
	// done survives the next start.
	bootAt(exe, "1.1.0", io.Discard)
	if st := bootState(); st.Phase != PhaseDone {
		t.Errorf("after restart: %+v", st)
	}
}

func TestBoot_WindowsRestartBecomesDone(t *testing.T) {
	exe, _ := stage(t, PhaseRestartRequired)
	bootAt(exe, "1.0.0", io.Discard) // not restarted yet
	if st := bootState(); st.Phase != PhaseRestartRequired {
		t.Errorf("old process: %+v", st)
	}
	bootAt(exe, "1.1.0", io.Discard)
	if st := bootState(); st.Phase != PhaseDone {
		t.Errorf("restarted: %+v", st)
	}
}

func TestRing_KeepsLastLines(t *testing.T) {
	r := &ring{}
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(r, "line %d\n", i)
	}
	s := r.String()
	if len(s) > logLimit || !strings.HasPrefix(s, "line ") || !strings.HasSuffix(s, "line 1999\n") {
		t.Errorf("ring = %d bytes, starts %q", len(s), s[:20])
	}
}

func TestAPI(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	h := setup(t, newRelease(t, priv, "1.1.0"), pub, ":memory:", nil)
	mux := http.NewServeMux()
	h.u.Register(mux, func(next http.Handler) http.Handler { return next })

	do := func(method, path, ctype, body string, hdr ...string) (*httptest.ResponseRecorder, map[string]any) {
		req := httptest.NewRequest(method, "http://127.0.0.1:8000"+path, strings.NewReader(body))
		if ctype != "" {
			req.Header.Set("Content-Type", ctype)
		}
		for i := 0; i+1 < len(hdr); i += 2 {
			req.Header.Set(hdr[i], hdr[i+1])
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var m map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &m)
		return rec, m
	}

	rec, info := do("GET", "/api/update", "", "")
	if rec.Code != 200 || info["latest"] != "1.1.0" || info["available"] != true || info["notes"] != "notes" || info["mode"] != "self" || info["memory_db"] != true {
		t.Errorf("GET: %d %v", rec.Code, info)
	}
	if st, _ := info["state"].(map[string]any); st["phase"] != "idle" {
		t.Errorf("GET state: %v", info["state"])
	}

	if rec, _ := do("POST", "/api/update", "application/json", `{"version":"1.1.0"}`, "Sec-Fetch-Site", "cross-site"); rec.Code != 403 {
		t.Errorf("cross-site: %d", rec.Code)
	}
	if rec, _ := do("POST", "/api/update", "text/plain", `{"version":"1.1.0"}`); rec.Code != 415 {
		t.Errorf("not JSON: %d", rec.Code)
	}
	if rec, m := do("POST", "/api/update", "application/json", `{"version":"1.0.9"}`); rec.Code != 400 || m["code"] != float64(400) {
		t.Errorf("not latest: %d %v", rec.Code, m)
	}

	if rec, _ := do("POST", "/api/update/check", "", ""); rec.Code != 200 {
		t.Errorf("check: %d", rec.Code)
	}
	if rec, m := do("POST", "/api/update/check", "", ""); rec.Code != 429 || m["retry_after"].(float64) < 1 {
		t.Errorf("second check: %d %v", rec.Code, m)
	}

	if rec, _ := do("POST", "/api/update", "application/json", `{"version":"1.1.0"}`); rec.Code != 202 {
		t.Errorf("start: %d", rec.Code)
	}
	if rec, _ := do("POST", "/api/update", "application/json", `{"version":"1.1.0"}`); rec.Code != 409 {
		t.Errorf("start while running: %d", rec.Code)
	}
	h.wait(t)

	h.u.mode = Mode{Mode: ModeManual, Reason: "set update.self_update: on (env UPDATE_SELF=on)"}
	if rec, m := do("POST", "/api/update", "application/json", `{"version":"1.1.0"}`); rec.Code != 403 || m["message"] != h.u.mode.Reason {
		t.Errorf("manual: %d %v", rec.Code, m)
	}
}

// The Host guard (DNS rebinding) applies only without auth on a loopback-only
// listen; an open port (update.self_update: on) answers by its LAN address.
func TestAPI_HostGuard(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	tests := []struct {
		name   string
		listen string
		auth   bool
		url    string
		want   int
	}{
		{"loopback listen, rebinding host", "127.0.0.1:8000", false, "http://evil.example:8000", 403},
		{"loopback listen, localhost", "127.0.0.1:8000", false, "http://localhost:8000", 202},
		{"open port, setting on, LAN host", ":8000", false, "http://192.168.1.5:8000", 202},
		{"auth on, any host", "127.0.0.1:8000", true, "http://bugs.example", 202},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := setup(t, newRelease(t, priv, "1.1.0"), pub, ":memory:", nil)
			h.u.cfg.HTTPAddr, h.u.cfg.AuthOn = tt.listen, tt.auth
			mux := http.NewServeMux()
			h.u.Register(mux, func(next http.Handler) http.Handler { return next })
			req := httptest.NewRequest("POST", tt.url+"/api/update", strings.NewReader(`{"version":"1.1.0"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Errorf("POST /api/update = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
			if rec.Code == 202 {
				h.wait(t)
			}
			chk := httptest.NewRequest("POST", tt.url+"/api/update/check", nil)
			rec = httptest.NewRecorder()
			mux.ServeHTTP(rec, chk)
			if wantCheck := map[bool]int{true: 403, false: 200}[tt.want == 403]; rec.Code != wantCheck {
				t.Errorf("POST /api/update/check = %d, want %d", rec.Code, wantCheck)
			}
		})
	}
}
