package app

import (
	"testing"
	"testing/fstest"
)

func TestCgroupMemoryLimit(t *testing.T) {
	for name, tc := range map[string]struct {
		files fstest.MapFS
		self  string
		want  int64
	}{
		"v2 limit":      {fstest.MapFS{"memory.max": {Data: []byte("536870912\n")}}, "/", 536870912},
		"v2 no limit":   {fstest.MapFS{"memory.max": {Data: []byte("max\n")}}, "/", 0},
		"v1 limit":      {fstest.MapFS{"memory/memory.limit_in_bytes": {Data: []byte("1073741824\n")}}, "/", 1073741824},
		"v1 no limit":   {fstest.MapFS{"memory/memory.limit_in_bytes": {Data: []byte("9223372036854771712\n")}}, "/", 0},
		"no cgroup":     {fstest.MapFS{}, "/", 0},
		"garbage":       {fstest.MapFS{"memory.max": {Data: []byte("lots")}}, "/", 0},
		"v2 own cgroup": {fstest.MapFS{"user.slice/run.scope/memory.max": {Data: []byte("300")}, "memory.max": {Data: []byte("max")}}, "/user.slice/run.scope", 300},
		"v2 before v1":  {fstest.MapFS{"memory.max": {Data: []byte("100")}, "memory/memory.limit_in_bytes": {Data: []byte("200")}}, "/", 100},
	} {
		if got := cgroupMemoryLimit(tc.files, tc.self); got != tc.want {
			t.Errorf("%s: %d, want %d", name, got, tc.want)
		}
	}
}

// GOMEMLIMIT set by the user wins over the cgroup limit.
func TestApplyMemoryLimit_EnvWins(t *testing.T) {
	t.Setenv("GOMEMLIMIT", "123MiB")
	if _, src := ApplyMemoryLimit(); src != "GOMEMLIMIT" {
		t.Errorf("source = %q", src)
	}
}

func TestSelfCgroup(t *testing.T) {
	if got := selfCgroup("12:memory:/x\n0::/user.slice/a.scope\n"); got != "/user.slice/a.scope" {
		t.Errorf("got %q", got)
	}
	if got := selfCgroup("12:memory:/x\n"); got != "" {
		t.Errorf("v1 only: %q", got)
	}
}
