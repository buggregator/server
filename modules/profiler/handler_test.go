package profiler

import (
	"encoding/json"
	"math"
	"net/http/httptest"
	"strings"
	"testing"
)

// xhprof_enable(XHPROF_FLAGS_MEMORY) without XHPROF_FLAGS_CPU sends no "cpu"
// key: the profile is still taken, CPU is 0 everywhere (GitHub #230).
func TestHandle_ProfileWithoutCPU(t *testing.T) {
	body := `{"app_name":"app","hostname":"h","date":1700000000,"tags":{},"profile":{
		"main()":{"wt":1000,"mu":2048,"pmu":4096,"ct":1},
		"main()==>foo()":{"wt":600,"mu":1024,"pmu":2048,"ct":5}}}`
	inc, err := (&handler{}).Handle(httptest.NewRequest("POST", "/api/profiler/store", strings.NewReader(body)))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	var p struct {
		Peaks Metrics         `json:"peaks"`
		Edges map[string]Edge `json:"edges"`
	}
	if err := json.Unmarshal(inc.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.Peaks.CPU != 0 || p.Peaks.WallTime != 1000 || p.Peaks.Memory != 2048 || len(p.Edges) != 2 {
		t.Errorf("peaks %+v, %d edges", p.Peaks, len(p.Edges))
	}
	for _, e := range p.Edges {
		if e.Cost.CPU != 0 || e.Diff.CPU != 0 || e.Percents.CPU != 0 || math.IsNaN(e.Percents.WallTime) {
			t.Errorf("edge %s: %+v", e.Callee, e)
		}
	}
}
