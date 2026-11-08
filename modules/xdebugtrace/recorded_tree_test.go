package xdebugtrace

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func recordedFile(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// entryOf is the offset of the first entry of fn in a recorded trace.
func entryOf(t *testing.T, raw []byte, fn string) int64 {
	t.Helper()
	tr, err := NewReader(bytes.NewReader(raw), 0)
	if err != nil {
		t.Fatal(err)
	}
	for {
		rec, err := tr.Next()
		if err == io.EOF {
			t.Fatalf("no entry of %s", fn)
		}
		if err != nil {
			t.Fatal(err)
		}
		if rec.Kind == Entry && rec.Func == fn {
			return tr.Offset()
		}
	}
}

func TestRecorded_MainTree(t *testing.T) {
	raw := recordedFile(t, "main.xt.gz")
	top, err := Children(bytes.NewReader(raw), 0, -1, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(top.Nodes) != 1 || top.Nodes[0].Func != "{main}" || top.Nodes[0].Children != 13 || !top.Nodes[0].Done || top.Truncated || top.Next != nil {
		t.Fatalf("top = %+v", top)
	}

	kids, err := Children(bytes.NewReader(raw), 0, top.Nodes[0].Offset, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"require", `App\UserRepo->__construct`, `App\UserRepo->find`, "strtoupper", `App\UserRepo->find`, "strtoupper", `App\UserRepo->find`, `App\fib`, "implode", "sprintf", "{closure:/app/main.php:14-14}", "Exception->__construct", "Exception->getMessage"}
	got := funcs(kids)
	if len(got) != len(want) {
		t.Fatalf("children = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("children = %v", got)
		}
	}
	if fib := kids.Nodes[7]; fib.Children != 2 || !fib.Done || fib.Return == nil || *fib.Return != "8" || fib.Args[0] != "6" {
		t.Errorf("fib = %+v", fib)
	}
	if kids.Parent.Children != 13 || kids.Truncated {
		t.Errorf("parent = %+v truncated=%v", kids.Parent, kids.Truncated)
	}

	cl, err := Children(bytes.NewReader(raw), 0, entryOf(t, raw, "{closure:/app/main.php:14-14}"), -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(cl.Nodes) != 1 || cl.Nodes[0].Func != "str_repeat" || *cl.Nodes[0].Return != `'a\tba\tb'` || *cl.Parent.Return != `'a\tba\tb'` {
		t.Errorf("closure = %+v", cl)
	}

	// Paging the 25 fib calls by 10.
	var all []Node
	from := int64(-1)
	for pages := 0; ; pages++ {
		p, err := Search(bytes.NewReader(raw), 0, "FIB", from, 10)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, p.Nodes...)
		if p.Truncated {
			t.Errorf("page %d truncated", pages)
		}
		if p.Next == nil {
			break
		}
		from = *p.Next
	}
	if len(all) != 25 || len(all[0].Path) != 1 || all[0].Path[0].Func != "{main}" {
		t.Fatalf("search: %d, first path %+v", len(all), all[0].Path)
	}
	for _, n := range all {
		if len(n.Path) != n.Level-1 || !n.Done || n.Return == nil {
			t.Errorf("match nr %d level %d: path %d done %v", n.Nr, n.Level, len(n.Path), n.Done)
		}
	}
}

// The trace started inside inner() (depth 4) and stopped inside after():
// the top level is inner's callee and after(), neither cut.
func TestRecorded_MidTree(t *testing.T) {
	raw := recordedFile(t, "mid.xt.gz")
	top, err := Children(bytes.NewReader(raw), 0, -1, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(top); len(got) != 2 || got[0] != `App\fib` || got[1] != "after" {
		t.Fatalf("top = %v", got)
	}
	if fib := top.Nodes[0]; fib.Level != 5 || fib.Children != 2 || !fib.Done || *fib.Return != "3" {
		t.Errorf("fib = %+v", fib)
	}
	if a := top.Nodes[1]; a.Level != 3 || a.Children != 2 || a.Done {
		t.Errorf("after = %+v", a)
	}
	if top.Truncated {
		t.Error("a trace stopped by xdebug_stop_trace() is not cut")
	}

	kids, err := Children(bytes.NewReader(raw), 0, top.Nodes[1].Offset, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(kids); len(got) != 2 || got[0] != "str_repeat" || got[1] != "xdebug_stop_trace" || *kids.Nodes[0].Return != "'yyy'" || kids.Truncated {
		t.Errorf("after's callees = %v %+v", got, kids)
	}

	s, err := Search(bytes.NewReader(raw), 0, "fib", -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Nodes) != 9 || len(s.Nodes[0].Path) != 0 || s.Truncated {
		t.Fatalf("search: %d truncated=%v", len(s.Nodes), s.Truncated)
	}
	for _, n := range s.Nodes {
		if len(n.Path) != n.Level-5 {
			t.Errorf("nr %d level %d: path %+v", n.Nr, n.Level, n.Path)
		}
	}
	st, err := Search(bytes.NewReader(raw), 0, "stop_trace", -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Nodes) != 1 || st.Nodes[0].Done || len(st.Nodes[0].Path) != 1 || st.Nodes[0].Path[0].Func != "after" || st.Truncated {
		t.Errorf("stop_trace: %+v truncated=%v", st.Nodes, st.Truncated)
	}
}
