package app

import (
	"io/fs"
	"math"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
)

// memLimitShare is the part of a container's memory limit given to the Go
// heap: the rest is left for goroutine stacks, the embedded PHP parser and
// SQLite's page cache.
const memLimitShare = 0.9

// cgroupMemoryLimit reads the memory limit of the process's cgroup from
// root, the /sys/fs/cgroup tree. The tightest limit wins, so every level from
// the process's own cgroup up to the root counts: a scope with no limit of its
// own under a limited slice is limited by the slice. v2 (memory.max, self from
// "0::/path" in /proc/self/cgroup; "/" inside a container) comes first; v1
// (memory/<path>/memory.limit_in_bytes, selfV1 from "N:memory:/path") only
// when there is no v2 file. 0 means no limit or none found.
func cgroupMemoryLimit(root fs.FS, self, selfV1 string) int64 {
	if limit, found := smallestLimit(root, "", self, "memory.max"); found {
		return limit
	}
	limit, _ := smallestLimit(root, "memory", selfV1, "memory.limit_in_bytes")
	return limit
}

// smallestLimit reads file in base/cgroup and every parent of it up to base and
// returns the smallest real limit; found tells whether any file existed.
func smallestLimit(root fs.FS, base, cgroup, file string) (limit int64, found bool) {
	dir := strings.Trim(cgroup, "/")
	for {
		name := path.Join(base, dir, file)
		if data, err := fs.ReadFile(root, name); err == nil {
			found = true
			if n := parseLimit(data); n > 0 && (limit == 0 || n < limit) {
				limit = n
			}
		}
		if dir == "" {
			return limit, found
		}
		if i := strings.LastIndex(dir, "/"); i >= 0 {
			dir = dir[:i]
		} else {
			dir = ""
		}
	}
}

// parseLimit reads a limit file: 0 for "max", garbage and v1's huge
// page-aligned "no limit" number.
func parseLimit(data []byte) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil || n <= 0 || n >= math.MaxInt64/2 {
		return 0
	}
	return n
}

// selfCgroup returns the v2 cgroup path ("0::/path") and the v1 memory
// controller's path ("N:...memory...:/path") from /proc/self/cgroup.
func selfCgroup(procSelfCgroup string) (v2, v1 string) {
	for _, line := range strings.Split(procSelfCgroup, "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[0] == "0" && parts[1] == "" {
			v2 = parts[2]
			continue
		}
		for _, c := range strings.Split(parts[1], ",") {
			if c == "memory" {
				v1 = parts[2]
			}
		}
	}
	return v2, v1
}

// ApplyMemoryLimit sets the Go runtime's soft memory limit, so the garbage
// collector works harder before a container's limit kills the process
// (GitHub #331). GOMEMLIMIT, when set, wins: the runtime has applied it
// already. Otherwise memLimitShare of the cgroup limit is used. It returns
// the limit in bytes (0 = none) and where it came from.
func ApplyMemoryLimit() (int64, string) {
	if os.Getenv("GOMEMLIMIT") != "" {
		// GOMEMLIMIT=off leaves no limit (math.MaxInt64): nothing to report.
		if limit := debug.SetMemoryLimit(-1); limit != math.MaxInt64 {
			return limit, "GOMEMLIMIT"
		}
		return 0, "GOMEMLIMIT"
	}
	data, _ := os.ReadFile("/proc/self/cgroup")
	v2, v1 := selfCgroup(string(data))
	limit := cgroupMemoryLimit(os.DirFS("/sys/fs/cgroup"), v2, v1)
	if limit == 0 {
		return 0, ""
	}
	soft := int64(float64(limit) * memLimitShare)
	debug.SetMemoryLimit(soft)
	return soft, "cgroup"
}
