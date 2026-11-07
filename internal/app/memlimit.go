package app

import (
	"io/fs"
	"math"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
)

// memLimitShare is the part of a container's memory limit given to the Go
// heap: the rest is left for goroutine stacks, the embedded PHP parser and
// SQLite's page cache.
const memLimitShare = 0.9

// cgroupMemoryLimit reads the memory limit of the cgroup the process runs in
// (v2 memory.max, else v1 memory.limit_in_bytes) from root, the
// /sys/fs/cgroup tree. 0 means no limit or none found.
func cgroupMemoryLimit(root fs.FS) int64 {
	for _, name := range []string{"memory.max", "memory/memory.limit_in_bytes"} {
		data, err := fs.ReadFile(root, name)
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(data))
		if v == "max" {
			return 0
		}
		n, err := strconv.ParseInt(v, 10, 64)
		// cgroup v1 reports "no limit" as a huge page-aligned number.
		if err != nil || n <= 0 || n >= math.MaxInt64/2 {
			return 0
		}
		return n
	}
	return 0
}

// ApplyMemoryLimit sets the Go runtime's soft memory limit, so the garbage
// collector works harder before a container's limit kills the process
// (GitHub #331). GOMEMLIMIT, when set, wins: the runtime has applied it
// already. Otherwise memLimitShare of the cgroup limit is used. It returns
// the limit in bytes (0 = none) and where it came from.
func ApplyMemoryLimit() (int64, string) {
	if os.Getenv("GOMEMLIMIT") != "" {
		return debug.SetMemoryLimit(-1), "GOMEMLIMIT"
	}
	limit := cgroupMemoryLimit(os.DirFS("/sys/fs/cgroup"))
	if limit == 0 {
		return 0, ""
	}
	soft := int64(float64(limit) * memLimitShare)
	debug.SetMemoryLimit(soft)
	return soft, "cgroup"
}
