package profiler

import (
	"testing"
)

func TestSplitEdgeName(t *testing.T) {
	tests := []struct {
		name       string
		wantCaller *string
		wantCallee string
	}{
		{"main()", nil, "main()"},
		{"main()==>foo()", ptr("main()"), "foo()"},
		{"foo()==>bar()==>baz()", ptr("foo()"), "bar()==>baz()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caller, callee := splitEdgeName(tt.name)
			if tt.wantCaller == nil {
				if caller != nil {
					t.Errorf("caller = %q, want nil", *caller)
				}
			} else {
				if caller == nil {
					t.Fatal("caller is nil, want non-nil")
				}
				if *caller != *tt.wantCaller {
					t.Errorf("caller = %q, want %q", *caller, *tt.wantCaller)
				}
			}
			if callee != tt.wantCallee {
				t.Errorf("callee = %q, want %q", callee, tt.wantCallee)
			}
		})
	}
}

func TestPct(t *testing.T) {
	tests := []struct {
		value, peak int64
		want        float64
	}{
		{100, 1000, 10.0},
		{0, 1000, 0},
		{100, 0, 0},
		{500, 500, 100.0},
	}

	for _, tt := range tests {
		got := pct(tt.value, tt.peak)
		if got != tt.want {
			t.Errorf("pct(%d, %d) = %f, want %f", tt.value, tt.peak, got, tt.want)
		}
	}
}

func TestMax64(t *testing.T) {
	if max64(10, 20) != 20 {
		t.Error("max64(10, 20) != 20")
	}
	if max64(30, 20) != 30 {
		t.Error("max64(30, 20) != 30")
	}
	if max64(-5, -10) != -5 {
		t.Error("max64(-5, -10) != -5")
	}
}

func TestProcess(t *testing.T) {
	t.Run("simple profile", func(t *testing.T) {
		incoming := &IncomingProfile{
			Profile: map[string]Metrics{
				"main()": {WallTime: 1000, CPU: 500, Memory: 2048, PeakMem: 4096, Calls: 1},
				"main()==>foo()": {WallTime: 600, CPU: 300, Memory: 1024, PeakMem: 2048, Calls: 5},
				"main()==>bar()": {WallTime: 200, CPU: 100, Memory: 512, PeakMem: 1024, Calls: 3},
				"foo()==>baz()":  {WallTime: 100, CPU: 50, Memory: 256, PeakMem: 512, Calls: 2},
			},
			AppName: "test-app",
		}

		peaks, edges := Process(incoming)

		// Peaks should be from main()
		if peaks.WallTime != 1000 {
			t.Errorf("peaks.WallTime = %d, want 1000", peaks.WallTime)
		}
		if peaks.CPU != 500 {
			t.Errorf("peaks.CPU = %d, want 500", peaks.CPU)
		}

		// Should have 4 edges
		if len(edges) != 4 {
			t.Errorf("len(edges) = %d, want 4", len(edges))
		}

		// Find the main() edge (no caller)
		var mainEdge *Edge
		for _, e := range edges {
			if e.Callee == "main()" {
				mainEdge = &e
				break
			}
		}
		if mainEdge == nil {
			t.Fatal("main() edge not found")
		}
		if mainEdge.Caller != nil {
			t.Error("main() should have nil caller")
		}

		// main() inclusive = 1000, children = 600+200 = 800, exclusive = 200
		if mainEdge.Diff.WallTime != 200 {
			t.Errorf("main() diff.WallTime = %d, want 200", mainEdge.Diff.WallTime)
		}

		// main() percentage should be 100%
		if mainEdge.Percents.WallTime != 100.0 {
			t.Errorf("main() percents.WallTime = %f, want 100.0", mainEdge.Percents.WallTime)
		}

		// Find foo() edge
		var fooEdge *Edge
		for _, e := range edges {
			if e.Callee == "foo()" {
				fooEdge = &e
				break
			}
		}
		if fooEdge == nil {
			t.Fatal("foo() edge not found")
		}
		if fooEdge.Caller == nil || *fooEdge.Caller != "main()" {
			t.Error("foo() should have main() as caller")
		}

		// foo() inclusive = 600, children = 100 (baz), exclusive = 500
		if fooEdge.Diff.WallTime != 500 {
			t.Errorf("foo() diff.WallTime = %d, want 500", fooEdge.Diff.WallTime)
		}

		// foo() percentage = 600/1000 * 100 = 60%
		if fooEdge.Percents.WallTime != 60.0 {
			t.Errorf("foo() percents.WallTime = %f, want 60.0", fooEdge.Percents.WallTime)
		}
	})

	t.Run("empty profile", func(t *testing.T) {
		incoming := &IncomingProfile{
			Profile: map[string]Metrics{},
		}
		peaks, edges := Process(incoming)
		if peaks.WallTime != 0 {
			t.Errorf("peaks should be zero")
		}
		if len(edges) != 0 {
			t.Errorf("edges should be empty")
		}
	})
}

func ptr(s string) *string {
	return &s
}
