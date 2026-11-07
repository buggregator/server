package app

import (
	"testing"
	"testing/fstest"
)

func TestCgroupMemoryLimit(t *testing.T) {
	for name, tc := range map[string]struct {
		files fstest.MapFS
		want  int64
	}{
		"v2 limit":     {fstest.MapFS{"memory.max": {Data: []byte("536870912\n")}}, 536870912},
		"v2 no limit":  {fstest.MapFS{"memory.max": {Data: []byte("max\n")}}, 0},
		"v1 limit":     {fstest.MapFS{"memory/memory.limit_in_bytes": {Data: []byte("1073741824\n")}}, 1073741824},
		"v1 no limit":  {fstest.MapFS{"memory/memory.limit_in_bytes": {Data: []byte("9223372036854771712\n")}}, 0},
		"no cgroup":    {fstest.MapFS{}, 0},
		"garbage":      {fstest.MapFS{"memory.max": {Data: []byte("lots")}}, 0},
		"v2 before v1": {fstest.MapFS{"memory.max": {Data: []byte("100")}, "memory/memory.limit_in_bytes": {Data: []byte("200")}}, 100},
	} {
		if got := cgroupMemoryLimit(tc.files); got != tc.want {
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
