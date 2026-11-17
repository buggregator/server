package selfupdate

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Phases of State.
const (
	PhaseIdle            = "idle"
	PhaseDownloading     = "downloading"
	PhaseVerifying       = "verifying"
	PhaseInstalling      = "installing"
	PhaseRestarting      = "restarting"
	PhaseDone            = "done"
	PhaseFailed          = "failed"
	PhaseRolledBack      = "rolled_back"
	PhaseRestartRequired = "restart_required"

	// phaseStarting is written by the new binary at boot and turns into done
	// once it serves; a boot that finds it means the last start died.
	phaseStarting = "starting"
)

// logLimit bounds the start-up output kept for a rolled_back state.
const logLimit = 8 << 10

// State is the progress of an update, sent to the UI as is.
type State struct {
	Phase      string     `json:"phase"`
	Version    string     `json:"version"`
	Previous   string     `json:"previous"`
	FailedStep string     `json:"failed_step"`
	Bytes      int64      `json:"bytes"`
	Total      int64      `json:"total"`
	Error      string     `json:"error"`
	Log        string     `json:"log"`
	UpdatedAt  *time.Time `json:"updated_at"`
}

// record is the state file next to the binary: the state plus what a
// rollback needs.
type record struct {
	State
	DB       string `json:"db,omitempty"`        // database file
	DBBackup string `json:"db_backup,omitempty"` // its copy taken before the swap
}

func statePath(exe string) string { return exe + ".update.json" }
func oldPath(exe string) string   { return exe + ".old" }

func readRecord(exe string) (record, bool) {
	var r record
	b, err := os.ReadFile(statePath(exe))
	if err != nil || json.Unmarshal(b, &r) != nil || r.Phase == "" {
		return r, false
	}
	return r, true
}

func writeRecord(exe string, r record) error {
	now := time.Now().UTC()
	r.UpdatedAt = &now
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	tmp := statePath(exe) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, statePath(exe))
}

// The process-wide boot state: set by Boot, read by the Updater, ended by
// Ready or Exit.
var boot struct {
	mu      sync.Mutex
	exe     string
	st      State
	pending *record // a boot after an update that has not served yet
	ring    *ring
}

var restarting atomic.Bool

// execFn replaces the process; tests swap it.
var execFn = execSelf

// Boot runs first in main. It reads the state an update left next to the
// binary: a first start of the new version keeps its output (for the
// rolled_back log) and is rolled back by Exit until Ready; a start that finds
// the last start of this version never got ready rolls back at once. The
// returned writer is where the log goes.
func Boot(version string) io.Writer {
	exe, err := executable()
	if err != nil {
		return os.Stdout
	}
	return bootAt(exe, version, os.Stdout)
}

func bootAt(exe, version string, out io.Writer) io.Writer {
	boot.mu.Lock()
	defer boot.mu.Unlock()
	boot.exe = exe
	boot.st = State{Phase: PhaseIdle}
	boot.pending = nil
	r, ok := readRecord(exe)
	if !ok {
		return out
	}
	switch {
	case r.Phase == PhaseRestarting && r.Version == version:
		r.Phase = phaseStarting
		_ = writeRecord(exe, r)
		boot.pending = &r
		boot.ring = &ring{}
		boot.st = r.State
		return io.MultiWriter(out, boot.ring)
	case r.Phase == phaseStarting && r.Version == version:
		// The last start of this version died before it served.
		if err := rollbackLocked(r, "the new version stopped before it was ready"); err != nil {
			slog.Error("self-update: rollback failed", "err", err)
		}
		r, _ = readRecord(exe)
	case (r.Phase == PhaseRestarting || r.Phase == phaseStarting) && r.Version != version:
		r.Phase, r.Error = PhaseRolledBack, "the new version did not start"
		_ = writeRecord(exe, r)
	case r.Phase == PhaseRestartRequired && r.Version == version:
		r.Phase = PhaseDone
		removeBackup(&r)
		_ = writeRecord(exe, r)
	}
	boot.st = r.State
	return out
}

// Ready marks a post-update boot as good: the state turns done and the
// database copy goes. Call it once the server serves.
func Ready() {
	boot.mu.Lock()
	defer boot.mu.Unlock()
	if boot.pending == nil {
		return
	}
	r := *boot.pending
	boot.pending = nil
	r.Phase, r.Error, r.Log = PhaseDone, "", ""
	removeBackup(&r)
	if err := writeRecord(boot.exe, r); err != nil {
		slog.Warn("self-update: cannot write state", "err", err)
	}
	boot.st = r.State
	slog.Info("self-update: done", "version", r.Version, "previous", r.Previous)
}

// Exit replaces os.Exit on start-up failures: a post-update boot that has not
// served yet restores the previous binary and database and runs it.
func Exit(code int) {
	rollbackPending()
	os.Exit(code)
}

func rollbackPending() {
	boot.mu.Lock()
	r := boot.pending
	boot.pending = nil
	if r != nil {
		if err := rollbackLocked(*r, "the new version failed to start"); err != nil {
			slog.Error("self-update: rollback failed", "err", err)
		}
	}
	boot.mu.Unlock()
}

// rollbackLocked puts the previous binary and database back, writes the
// rolled_back state and runs the previous binary. It returns only on error.
func rollbackLocked(r record, reason string) error {
	exe := boot.exe
	if _, err := os.Stat(oldPath(exe)); err != nil {
		return errors.New("no previous binary to roll back to")
	}
	if err := os.Rename(oldPath(exe), exe); err != nil {
		return err
	}
	if r.DBBackup != "" && r.DB != "" {
		if err := copyFile(r.DBBackup, r.DB); err != nil {
			slog.Error("self-update: cannot restore the database copy", "backup", r.DBBackup, "err", err)
		} else {
			os.Remove(r.DB + "-wal")
			os.Remove(r.DB + "-shm")
			removeBackup(&r)
		}
	}
	r.Phase, r.Error, r.FailedStep = PhaseRolledBack, reason, "starting"
	if boot.ring != nil {
		r.Log = boot.ring.String()
	}
	if err := writeRecord(exe, r); err != nil {
		return err
	}
	slog.Warn("self-update: rolled back", "to", r.Previous, "reason", reason)
	return execFn(exe)
}

func removeBackup(r *record) {
	if r.DBBackup != "" {
		os.Remove(r.DBBackup)
		r.DBBackup = ""
	}
}

func bootState() State {
	boot.mu.Lock()
	defer boot.mu.Unlock()
	if boot.st.Phase == "" || boot.st.Phase == phaseStarting {
		return State{Phase: PhaseIdle}
	}
	return boot.st
}

// Restarting reports that an update asked for a restart: main calls Reexec
// once everything is closed.
func Restarting() bool { return restarting.Load() }

// Reexec runs the binary in place of this process (the new one after a swap).
func Reexec() {
	exe := boot.exe
	if exe == "" {
		exe, _ = executable()
	}
	slog.Info("self-update: restarting", "exe", exe)
	if err := execFn(exe); err != nil {
		slog.Error("self-update: restart failed, start the server again", "err", err)
		os.Exit(1)
	}
}

// ring keeps the last logLimit bytes written.
type ring struct {
	mu  sync.Mutex
	buf []byte
}

func (r *ring) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.buf = append(r.buf, p...)
	if over := len(r.buf) - logLimit; over > 0 {
		r.buf = append(r.buf[:0], r.buf[over:]...)
	}
	return len(p), nil
}

// String is the kept output from the first whole line.
func (r *ring) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := string(r.buf)
	if len(r.buf) >= logLimit {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		}
	}
	return s
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".restore"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
