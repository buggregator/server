// Package selfupdate updates a binary install from the UI: download the
// release asset, check checksums.txt and its ed25519 signature, probe the new
// binary, swap it in keeping the old one, restart, and roll back when the new
// one does not start. Containers, read-only installs and open ports without
// auth get the commands to run instead.
package selfupdate

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/buggregator/server/v2/internal/updates"
)

const (
	maxAsset     = 512 << 20
	maxSmall     = 1 << 20
	probeTimeout = 15 * time.Second
	pushEvery    = 250 * time.Millisecond // ≤4 progress pushes a second
	checkEvery   = time.Minute
)

// Errors of Start, mapped to HTTP codes by the API.
var (
	ErrBusy       = errors.New("an update is already running")
	ErrNotLatest  = errors.New("only the latest release can be installed")
	ErrNotAllowed = errors.New("update from the UI is not allowed")
)

// Config wires an Updater.
type Config struct {
	Checker   *updates.Checker // nil: the update check is off
	Setting   string           // update.self_update
	AuthOn    bool
	HTTPAddr  string
	DSN       string
	DB        *sql.DB
	Broadcast func(State) // sends update.progress; may be nil
	Restart   func()      // stops the app so main re-execs; may be nil
}

// Info is the body of GET /api/update.
type Info struct {
	Current       string     `json:"current"`
	Latest        string     `json:"latest"`
	Available     bool       `json:"available"`
	URL           string     `json:"url"`
	Notes         string     `json:"notes"`
	CheckedAt     *time.Time `json:"checked_at"`
	CheckEnabled  bool       `json:"check_enabled"`
	Mode          string     `json:"mode"`
	Reason        string     `json:"reason"`
	Install       string     `json:"install"`
	OS            string     `json:"os"`
	Commands      []Command  `json:"commands"`
	MemoryDB      bool       `json:"memory_db"`
	RestartManual bool       `json:"restart_manual"`
	State         State      `json:"state"`
}

// Updater runs one update at a time.
type Updater struct {
	cfg    Config
	mode   Mode
	exe    string
	goos   string
	goarch string
	key    ed25519.PublicKey
	client *http.Client

	mu        sync.Mutex
	st        State
	running   bool
	lastPush  time.Time
	lastCheck time.Time
}

// New decides the mode for this process.
func New(cfg Config, version string) *Updater {
	exe, err := executable()
	f := Facts{
		Version:   version,
		Release:   updates.IsRelease(version),
		CheckOn:   cfg.Checker != nil,
		Setting:   cfg.Setting,
		AuthOn:    cfg.AuthOn,
		HTTPAddr:  cfg.HTTPAddr,
		GOOS:      runtime.GOOS,
		Exe:       exe,
		Container: container(),
		Writable:  err == nil && writable(filepath.Dir(exe)),
	}
	u := newUpdater(cfg, Decide(f), exe, runtime.GOOS, runtime.GOARCH)
	if u.mode.Mode != ModeOff {
		slog.Info("self-update", "mode", u.mode.Mode, "install", u.mode.Install, "reason", u.mode.Reason)
	}
	return u
}

func newUpdater(cfg Config, m Mode, exe, goos, goarch string) *Updater {
	return &Updater{
		cfg: cfg, mode: m, exe: exe, goos: goos, goarch: goarch,
		key:    publicKey(),
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// Info is the full answer of GET /api/update.
func (u *Updater) Info() Info {
	in := Info{
		Mode: u.mode.Mode, Reason: u.mode.Reason, Install: u.mode.Install, OS: u.goos,
		RestartManual: u.mode.RestartManual,
		MemoryDB:      dbFile(u.cfg.DSN) == "",
		CheckEnabled:  u.cfg.Checker != nil,
	}
	if c := u.cfg.Checker; c != nil {
		st := c.Status()
		in.Current, in.Latest, in.Available, in.URL, in.CheckedAt = st.Current, st.Latest, st.Available, st.URL, st.CheckedAt
		in.Notes = c.Release().Notes
	}
	if in.Mode != ModeOff {
		in.Commands = Commands(in.Install, u.goos, u.exe, in.Latest)
	}
	if in.Commands == nil {
		in.Commands = []Command{}
	}
	u.mu.Lock()
	in.State = u.st
	u.mu.Unlock()
	if in.State.Phase == "" {
		// Nothing started in this process: what the last update left (done
		// once Ready ran, rolled_back, …).
		in.State = bootState()
	}
	return in
}

// CheckNow asks GitHub at once, at most once a minute; wait > 0 is how long
// until the next check is allowed.
func (u *Updater) CheckNow(ctx context.Context) (wait time.Duration, err error) {
	if u.cfg.Checker == nil {
		return 0, errors.New("the update check is off")
	}
	u.mu.Lock()
	if since := time.Since(u.lastCheck); !u.lastCheck.IsZero() && since < checkEvery {
		u.mu.Unlock()
		return checkEvery - since, nil
	}
	u.lastCheck = time.Now()
	u.mu.Unlock()
	return 0, u.cfg.Checker.CheckNow(ctx)
}

// Start begins the update to version in the background.
func (u *Updater) Start(version string) error {
	if u.mode.Mode != ModeSelf {
		return fmt.Errorf("%w: %s", ErrNotAllowed, u.mode.Reason)
	}
	st := u.cfg.Checker.Status()
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.running {
		return ErrBusy
	}
	if !st.Available || version != st.Latest {
		return ErrNotLatest
	}
	u.running = true
	u.setLocked(State{Phase: PhaseDownloading, Version: version, Previous: st.Current})
	go u.run(version, st.Current, u.cfg.Checker.Release())
	return nil
}

// AssetName is the release file of this platform.
func AssetName(goos, goarch string) string {
	name := "buggregator-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func (u *Updater) run(version, current string, rel updates.Release) {
	ctx := context.Background()
	tmp, step, err := u.fetch(ctx, version, rel)
	if err == nil {
		step, err = u.install(ctx, tmp, version, current)
	}
	if err != nil {
		if tmp != "" {
			os.Remove(tmp)
		}
		slog.Warn("self-update failed", "step", step, "err", err)
		u.set(func(s *State) { s.Phase, s.FailedStep, s.Error = PhaseFailed, step, err.Error() }, true)
	}
}

// fetch downloads and verifies the new binary into a temp file next to the
// running one.
func (u *Updater) fetch(ctx context.Context, version string, rel updates.Release) (tmp, step string, err error) {
	name := AssetName(u.goos, u.goarch)
	binURL, sumsURL, sigURL := rel.Assets[name], rel.Assets["checksums.txt"], rel.Assets["checksums.txt.sig"]
	switch {
	case binURL == "":
		return "", PhaseDownloading, errors.New("the release has no file " + name)
	case sumsURL == "":
		return "", PhaseVerifying, errors.New("the release has no checksums.txt")
	case sigURL == "":
		return "", PhaseVerifying, errNotSigned
	}

	f, err := os.CreateTemp(filepath.Dir(u.exe), ".buggregator-"+version+"-*")
	if err != nil {
		return "", PhaseDownloading, err
	}
	tmp = f.Name()
	h := sha256.New()
	err = u.download(ctx, binURL, io.MultiWriter(f, h), true)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return tmp, PhaseDownloading, err
	}

	u.set(func(s *State) { s.Phase = PhaseVerifying }, true)
	var sums, sig bytes.Buffer
	if err := u.download(ctx, sumsURL, &sums, false); err != nil {
		return tmp, PhaseVerifying, fmt.Errorf("checksums.txt: %w", err)
	}
	if err := u.download(ctx, sigURL, &sig, false); err != nil {
		return tmp, PhaseVerifying, fmt.Errorf("checksums.txt.sig: %w", err)
	}
	if err := verifySignature(u.key, sums.Bytes(), sig.Bytes()); err != nil {
		return tmp, PhaseVerifying, err
	}
	want, err := checksumFor(sums.Bytes(), name)
	if err != nil {
		return tmp, PhaseVerifying, err
	}
	if !bytes.Equal(want, h.Sum(nil)) {
		return tmp, PhaseVerifying, errors.New("the sha256 of " + name + " does not match checksums.txt")
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return tmp, PhaseVerifying, err
	}
	if err := probe(ctx, tmp, version); err != nil {
		return tmp, PhaseVerifying, err
	}
	return tmp, "", nil
}

// install copies the database, swaps the binaries and restarts.
func (u *Updater) install(ctx context.Context, tmp, version, current string) (string, error) {
	u.set(func(s *State) { s.Phase = PhaseInstalling }, true)
	r := record{DB: dbFile(u.cfg.DSN)}
	if r.DB != "" && !u.mode.RestartManual && u.cfg.DB != nil {
		r.DBBackup = r.DB + ".pre-" + current + ".bak"
		os.Remove(r.DBBackup)
		if _, err := u.cfg.DB.ExecContext(ctx, "VACUUM INTO ?", r.DBBackup); err != nil {
			return PhaseInstalling, fmt.Errorf("database copy: %w", err)
		}
	}
	old := oldPath(u.exe)
	os.Remove(old)
	if err := os.Rename(u.exe, old); err != nil {
		removeBackup(&r)
		return PhaseInstalling, err
	}
	if err := os.Rename(tmp, u.exe); err != nil {
		_ = os.Rename(old, u.exe)
		removeBackup(&r)
		return PhaseInstalling, err
	}
	phase := PhaseRestarting
	if u.mode.RestartManual {
		phase = PhaseRestartRequired
	}
	r.State = State{Phase: phase, Version: version, Previous: current}
	if err := writeRecord(u.exe, r); err != nil {
		slog.Warn("self-update: cannot write state", "err", err)
	}
	u.set(func(s *State) { s.Phase = phase }, true)
	if phase == PhaseRestarting {
		slog.Info("self-update: installed, restarting", "version", version, "previous", current)
		restarting.Store(true)
		if u.cfg.Restart != nil {
			u.cfg.Restart()
		}
	}
	return "", nil
}

// download copies url to w; progress updates the state.
func (u *Updater) download(ctx context.Context, url string, w io.Writer, progress bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "buggregator-self-update")
	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	limit := int64(maxSmall)
	if progress {
		limit = maxAsset
		u.set(func(s *State) { s.Total = resp.ContentLength }, false)
	}
	var src io.Reader = io.LimitReader(resp.Body, limit+1)
	if progress {
		src = &counter{r: src, add: func(n int64) { u.set(func(s *State) { s.Bytes += n }, false) }}
	}
	n, err := io.Copy(w, src)
	if err != nil {
		return err
	}
	if n > limit {
		return fmt.Errorf("GET %s: larger than %d bytes", url, limit)
	}
	if progress {
		u.set(func(s *State) {}, true)
	}
	return nil
}

// probe runs the new binary with --version: it must start on this machine
// and be the version it claims.
func probe(ctx context.Context, bin, version string) error {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return fmt.Errorf("the new binary does not run: %w", err)
	}
	if got := strings.TrimPrefix(strings.TrimSpace(string(out)), "v"); got != version {
		return fmt.Errorf("the new binary reports version %q, want %q", got, version)
	}
	return nil
}

// set changes the state and pushes it: always when force, else at most
// every pushEvery (download progress).
func (u *Updater) set(change func(*State), force bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	s := u.st
	change(&s)
	// Only a failure frees the updater: after a swap the process restarts
	// (or waits for a manual restart on Windows).
	if s.Phase == PhaseFailed {
		u.running = false
	}
	if !force && time.Since(u.lastPush) < pushEvery {
		u.st = s
		return
	}
	u.setLocked(s)
}

func (u *Updater) setLocked(s State) {
	now := time.Now().UTC()
	s.UpdatedAt = &now
	u.st = s
	u.lastPush = time.Now()
	if u.cfg.Broadcast != nil {
		u.cfg.Broadcast(s)
	}
}

type counter struct {
	r   io.Reader
	add func(int64)
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.add(int64(n))
	}
	return n, err
}

// dbFile is the database file of a DSN; "" for an in-memory database.
func dbFile(dsn string) string {
	s := strings.TrimPrefix(dsn, "file:")
	if i := strings.IndexByte(s, '?'); i >= 0 {
		if strings.Contains(s[i:], "mode=memory") {
			return ""
		}
		s = s[:i]
	}
	if s == "" || strings.HasPrefix(s, ":memory:") {
		return ""
	}
	return s
}
