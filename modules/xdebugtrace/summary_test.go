package xdebugtrace

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSummarize(t *testing.T) {
	for name, in := range map[string]string{"plain": sample, "gzip": string(gz(t, sample))} {
		t.Run(name, func(t *testing.T) {
			s, err := Summarize(strings.NewReader(in), 0)
			if err != nil {
				t.Fatal(err)
			}
			if s.Calls != 6 || s.UserCalls != 4 || s.MaxDepth != 4 || s.Assignments != 1 || !s.Returns || s.Truncated || s.Main != "/app/index.php" {
				t.Errorf("counts: %+v", s)
			}
			if !near(s.Duration, 0.0102) || s.PeakMemory != 452000 || s.Version != "3.4.5" {
				t.Errorf("duration/memory: %v %v", s.Duration, s.PeakMemory)
			}
			// PDO->query: 0.0027 → 0.0097 = 7 ms, all self.
			if s.Top[0].Func != "PDO->query" || !near(s.Top[0].Self, 0.007) || s.Top[0].User {
				t.Errorf("top[0]: %+v", s.Top[0])
			}
			self := map[string]float64{}
			for _, f := range s.Top {
				self[f.Func] = f.Self
			}
			// Db->query 7.2 ms incl − 7 ms PDO = 0.2 ms self; include 2 ms self.
			if !near(self[`App\Db->query`], 0.0002) || !near(self["include"], 0.002) {
				t.Errorf("self: %v", self)
			}
			if s.Slowest[0].Func != "{main}" || !near(s.Slowest[0].Incl, 0.01) || s.Slowest[0].Offset != int64(strings.Index(sample, "1\t0\t0\t")) {
				t.Errorf("slowest[0]: %+v", s.Slowest[0])
			}
			if len(s.Slowest) != 6 || s.Slowest[1].Func != `App\Kernel->handle` {
				t.Errorf("slowest order: %+v", s.Slowest)
			}
			var mem int64
			for _, c := range s.Slowest {
				if c.Func == "include" {
					mem = c.Memory
				}
			}
			if mem != 47000 {
				t.Errorf("include memory delta: %d", mem)
			}
			if len(s.Index) != 1 || s.Index[0].Nr != 0 {
				t.Errorf("index: %+v", s.Index)
			}
		})
	}
}

// A trace cut by the size limit, or with no TRACE END (fatal error, killed
// worker), is summarized up to where it stops and flagged.
func TestSummarize_Truncated(t *testing.T) {
	s, err := Summarize(strings.NewReader(sample), 400)
	if err != nil || !s.Truncated || s.Calls == 0 || s.Calls == 6 {
		t.Fatalf("limit: %v %+v", err, s)
	}

	noEnd := sample[:strings.Index(sample, "4\t5\t1\t")]
	s, err = Summarize(strings.NewReader(noEnd), 0)
	if err != nil || !s.Truncated || s.Calls != 6 {
		t.Fatalf("no end: %v %+v", err, s)
	}
	for _, c := range s.Slowest {
		if c.Incl < 0 || c.Self < 0 {
			t.Errorf("negative time: %+v", c)
		}
	}

	broken := gz(t, sample)
	s, err = Summarize(strings.NewReader(string(broken[:len(broken)-20])), 0)
	if err != nil || !s.Truncated {
		t.Fatalf("broken gzip tail: %v %+v", err, s)
	}
}

// An exit for an outer call closes frames left open inside it (an exception
// unwound them without their own exit lines).
func TestSummarize_UnwoundFrames(t *testing.T) {
	in := "File format: 4\n" +
		"1\t0\t0\t0.0\t100\t{main}\t1\t\t/a.php\t0\t0\n" +
		"2\t1\t0\t0.1\t100\tf\t1\t\t/a.php\t3\t0\n" +
		"3\t2\t0\t0.2\t100\tg\t1\t\t/a.php\t5\t0\n" +
		"1\t0\t1\t0.5\t100\n" +
		"TRACE END   [x]\n"
	s, err := Summarize(strings.NewReader(in), 0)
	if err != nil || s.Truncated {
		t.Fatal(err, s.Truncated)
	}
	incl := map[string]float64{}
	for _, c := range s.Slowest {
		incl[c.Func] = c.Incl
	}
	if !near(incl["g"], 0.3) || !near(incl["f"], 0.4) || !near(incl["{main}"], 0.5) {
		t.Errorf("incl: %v", incl)
	}
}

// Many calls: top lists stay at TopN, the index has a point every IndexEvery
// calls, memory stays bounded by the stack depth.
func TestSummarize_Large(t *testing.T) {
	var b strings.Builder
	b.WriteString("File format: 4\n1\t0\t0\t0\t1\t{main}\t1\t\t/a.php\t0\t0\n")
	n := 5*IndexEvery + 7
	tm := 0.0
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "2\t%d\t0\t%f\t1\tfn%d\t1\t\t/a.php\t%d\t0\n", i, tm, i%50, i)
		tm += float64(i%13) * 1e-6
		fmt.Fprintf(&b, "2\t%d\t1\t%f\t1\n", i, tm)
	}
	fmt.Fprintf(&b, "1\t0\t1\t%f\t1\nTRACE END   [x]\n", tm)
	s, err := Summarize(strings.NewReader(b.String()), 0)
	if err != nil {
		t.Fatal(err)
	}
	if s.Calls != n+1 || len(s.Top) != TopN || len(s.Slowest) != TopN || len(s.Index) != 6 {
		t.Fatalf("calls %d top %d slowest %d index %d", s.Calls, len(s.Top), len(s.Slowest), len(s.Index))
	}
	for i := 1; i < len(s.Slowest); i++ {
		if s.Slowest[i].Incl > s.Slowest[i-1].Incl {
			t.Fatalf("slowest not sorted at %d", i)
		}
	}
	text := b.String()
	for _, p := range s.Index {
		if !strings.HasPrefix(text[p.Offset:], fmt.Sprintf("%d\t%d\t0\t", p.Level, p.Nr)) {
			t.Errorf("index point %+v does not start its entry line", p)
		}
	}
}
