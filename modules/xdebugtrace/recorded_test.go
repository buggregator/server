package xdebugtrace

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// Traces recorded on a live Xdebug 3.4.5; see testdata/README.md.

func readRecorded(t *testing.T, name string) ([]Record, Header, []byte) {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewReader(bytes.NewReader(raw), 0)
	if err != nil {
		t.Fatal(err)
	}
	var recs []Record
	for {
		rec, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if rec.Kind == Entry {
			if line := string(plain[r.Offset():]); !strings.HasPrefix(line, fmt.Sprintf("%d\t%d\t0\t", rec.Level, rec.Nr)) {
				t.Errorf("nr %d: offset %d points at %q", rec.Nr, r.Offset(), line[:min(30, len(line))])
			}
		}
		recs = append(recs, rec)
	}
	return recs, r.Header, plain
}

func find(recs []Record, kind Kind, fn string) *Record {
	for i := range recs {
		if recs[i].Kind == kind && recs[i].Func == fn {
			return &recs[i]
		}
	}
	return nil
}

func countKinds(recs []Record) map[Kind]int {
	n := map[Kind]int{}
	for _, r := range recs {
		n[r.Kind]++
	}
	return n
}

func TestRecorded_MainRecords(t *testing.T) {
	recs, h, _ := readRecorded(t, "main.xt.gz")
	if h.Version != "3.4.5" || h.FileFormat != 4 || h.Start != "2026-11-08 17:24:21.055261" || h.End != "2026-11-08 17:24:21.055572" || h.EndTime != 0.00143 || h.EndMemory != 530336 {
		t.Errorf("header: %+v", h)
	}
	if n := countKinds(recs); n[Entry] != 39 || n[Exit] != 39 || n[Return] != 36 || n[Assign] != 24 {
		t.Errorf("kinds: %v", n)
	}
	if m := recs[0]; m.Kind != Entry || m.Func != "{main}" || m.Level != 1 || m.Nr != 0 || !m.User || m.File != "/app/main.php" || m.Line != 0 || len(m.Args) != 0 {
		t.Errorf("{main}: %+v", m)
	}
	if rq := find(recs, Entry, "require"); rq == nil || rq.Include != "/app/lib.php" || rq.File != "/app/main.php" || rq.Line != 2 || rq.Level != 2 {
		t.Errorf("require: %+v", rq)
	}
	// The first assignment is inside the constructor, in the required file.
	for _, a := range recs {
		if a.Kind == Assign {
			if a.Level != 2 || a.File != "/app/lib.php" || a.Line != 9 || a.Value != "$i = 1" {
				t.Errorf("assign: %+v", a)
			}
			break
		}
	}
	// Xdebug escapes tab and newline inside values, so a value never splits.
	up := find(recs, Entry, "strtoupper")
	if up == nil || up.User || len(up.Args) != 1 || up.Args[0] != `'user\t1\nline'` {
		t.Fatalf("strtoupper: %+v", up)
	}
	for _, r := range recs {
		if r.Kind == Return && r.Nr == up.Nr && r.Value != `'USER\t1\nLINE'` {
			t.Errorf("return: %+v", r)
		}
	}
	if sp := find(recs, Entry, "sprintf"); sp == nil || len(sp.Args) != 3 || sp.Args[0] != `'%s|%d'` || sp.Args[1] != `'USER\t1\nLINE,USER\t2\nLINE,missing'` || sp.Args[2] != "8" {
		t.Errorf("sprintf: %+v", sp)
	}
	if c := find(recs, Entry, "{closure:/app/main.php:14-14}"); c == nil || !c.User || c.Line != 15 || c.Args[0] != `'a\tb'` {
		t.Errorf("closure: %+v", c)
	}
	if e := find(recs, Entry, "Exception->__construct"); e == nil || e.User || e.Args[0] != "'boom'" {
		t.Errorf("exception: %+v", e)
	}
}

func TestRecorded_MainSummary(t *testing.T) {
	f, _ := os.Open("testdata/main.xt.gz")
	defer f.Close()
	s, err := Summarize(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s.Calls != 39 || s.UserCalls != 32 || s.MaxDepth != 7 || s.Assignments != 24 || !s.Returns || s.Truncated || s.Main != "/app/main.php" || s.Duration != 0.00143 || s.PeakMemory != 542304 {
		t.Errorf("summary: %+v", s)
	}
	counts := map[string]int{}
	for _, fs := range s.Top {
		counts[fs.Func] = fs.Count
	}
	if counts[`App\fib`] != 25 || counts[`App\UserRepo->find`] != 3 || counts["strtoupper"] != 2 {
		t.Errorf("top: %v", counts)
	}
	if sl := s.Slowest[0]; sl.Func != "{main}" || fmt.Sprintf("%.6f", sl.Incl) != "0.000272" {
		t.Errorf("slowest: %+v", sl)
	}
	if len(s.Index) != 1 || s.Index[0].Nr != 0 || s.Index[0].Level != 1 {
		t.Errorf("index: %+v", s.Index)
	}
}

// A trace started by xdebug_start_trace() deep in the stack: it begins at
// level 5, exits of frames it never saw enter, and stops inside
// xdebug_stop_trace() with no exit line.
func TestRecorded_MidTrace(t *testing.T) {
	recs, h, _ := readRecorded(t, "mid.xt.gz")
	if h.End == "" || h.EndTime != 0.000275 || h.EndMemory != 536296 {
		t.Errorf("header: %+v", h)
	}
	if r := recs[0]; r.Kind != Entry || r.Level != 5 || r.Func != `App\fib` || r.File != "/app/mid.php" || r.Line != 8 || r.Args[0] != "4" {
		t.Errorf("first: %+v", r)
	}
	if n := countKinds(recs); n[Entry] != 12 || n[Exit] != 12 || n[Return] != 10 || n[Assign] != 2 {
		t.Errorf("kinds: %v", n)
	}
	if a := find(recs, Entry, "after"); a == nil || a.Level != 3 || a.Line != 4 {
		t.Errorf("after: %+v", a)
	}

	f, _ := os.Open("testdata/mid.xt.gz")
	defer f.Close()
	s, err := Summarize(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s.Calls != 12 || s.UserCalls != 10 || s.MaxDepth != 8 || s.Truncated || s.Main != "" || s.Duration != 0.000275 || s.PeakMemory != 536320 {
		t.Errorf("summary: %+v", s)
	}
	var fib, stop *FuncStat
	for i := range s.Top {
		switch s.Top[i].Func {
		case `App\fib`:
			fib = &s.Top[i]
		case "xdebug_stop_trace":
			stop = &s.Top[i]
		}
	}
	if fib == nil || fib.Count != 9 || stop == nil || stop.Count != 1 {
		t.Errorf("top: fib %+v stop %+v", fib, stop)
	}
}
