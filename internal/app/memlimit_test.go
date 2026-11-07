package app

import (
	"testing"
	"testing/fstest"
)

func file(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }

func TestCgroupMemoryLimit(t *testing.T) {
	for name, tc := range map[string]struct {
		files    fstest.MapFS
		self, v1 string
		want     int64
	}{
		"v2 limit":     {fstest.MapFS{"memory.max": file("536870912\n")}, "/", "", 536870912},
		"v2 no limit":  {fstest.MapFS{"memory.max": file("max\n")}, "/", "", 0},
		"v1 limit":     {fstest.MapFS{"memory/memory.limit_in_bytes": file("1073741824\n")}, "", "/", 1073741824},
		"v1 no limit":  {fstest.MapFS{"memory/memory.limit_in_bytes": file("9223372036854771712\n")}, "", "/", 0},
		"no cgroup":    {fstest.MapFS{}, "/", "", 0},
		"garbage":      {fstest.MapFS{"memory.max": file("lots")}, "/", "", 0},
		"v2 before v1": {fstest.MapFS{"memory.max": file("100"), "memory/memory.limit_in_bytes": file("200")}, "/", "/", 100},
		"v2 own cgroup": {fstest.MapFS{"user.slice/run.scope/memory.max": file("300"), "user.slice/memory.max": file("max"), "memory.max": file("max")},
			"/user.slice/run.scope", "", 300},
		"v2 limited parent": {fstest.MapFS{"sys.slice/app.service/memory.max": file("max"), "sys.slice/memory.max": file("400")},
			"/sys.slice/app.service", "", 400},
		"v2 smallest wins": {fstest.MapFS{"a/b/memory.max": file("900"), "a/memory.max": file("500"), "memory.max": file("700")},
			"/a/b", "", 500},
		"v1 own cgroup": {fstest.MapFS{"memory/docker/x/memory.limit_in_bytes": file("600"), "memory/memory.limit_in_bytes": file("9223372036854771712")},
			"", "/docker/x", 600},
	} {
		if got := cgroupMemoryLimit(tc.files, tc.self, tc.v1); got != tc.want {
			t.Errorf("%s: %d, want %d", name, got, tc.want)
		}
	}
}

// GOMEMLIMIT set by the user wins over the cgroup limit; "off" reports none.
func TestApplyMemoryLimit_EnvWins(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "off")
	if limit, src := ApplyMemoryLimit(); src != "GOMEMLIMIT" || limit != 0 {
		t.Errorf("limit %d, source %q", limit, src)
	}
}

func TestSelfCgroup(t *testing.T) {
	if v2, v1 := selfCgroup("12:memory:/x\n0::/user.slice/a.scope\n"); v2 != "/user.slice/a.scope" || v1 != "/x" {
		t.Errorf("hybrid: %q %q", v2, v1)
	}
	if v2, v1 := selfCgroup("5:cpu,memory:/docker/abc\n"); v2 != "" || v1 != "/docker/abc" {
		t.Errorf("v1 only: %q %q", v2, v1)
	}
}
