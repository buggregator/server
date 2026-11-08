package xdebugtrace

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const tempPrefix = "buggregator-xdebug-traces-"

// TempDir makes the trace dir for memory storage mode: <base>/
// buggregator-xdebug-traces-<pid>. Dirs of processes that are gone (a
// crash, kill -9 skips the cleanup at exit) are removed first, so stale
// traces never pile up. base "" = os.TempDir().
func TempDir(base string) (string, error) {
	if base == "" {
		base = os.TempDir()
	}
	self := os.Getpid()
	if entries, err := os.ReadDir(base); err == nil {
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() || !strings.HasPrefix(name, tempPrefix) {
				continue
			}
			pid, err := strconv.Atoi(strings.TrimPrefix(name, tempPrefix))
			if err != nil || pid <= 0 || (pid != self && processAlive(pid)) {
				continue
			}
			// Our own pid here is a leftover of an earlier process.
			if err := os.RemoveAll(filepath.Join(base, name)); err == nil {
				slog.Info("xdebug-trace: removed stale temp dir", "dir", name)
			}
		}
	}
	dir := filepath.Join(base, tempPrefix+strconv.Itoa(self))
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
