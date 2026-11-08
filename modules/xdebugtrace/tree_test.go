package xdebugtrace

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// entryOffsets maps a function name of the sample to the offset of its entry.
func entryOffsets(t *testing.T) map[string]int64 {
	t.Helper()
	tr, err := NewReader(bytes.NewReader([]byte(sample)), 0)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]int64{}
	for {
		rec, err := tr.Next()
		if err == io.EOF {
			return m
		}
		if err != nil {
			t.Fatal(err)
		}
		if rec.Kind == Entry {
			m[rec.Func] = tr.Offset()
		}
	}
}

func funcs(p *Page) []string {
	var out []string
	for _, n := range p.Nodes {
		out = append(out, n.Func)
	}
	return out
}

func TestChildren(t *testing.T) {
	off := entryOffsets(t)
	for name, in := range map[string][]byte{"plain": []byte(sample), "gzip": gz(t, sample)} {
		t.Run(name, func(t *testing.T) {
			open := func() io.Reader { return bytes.NewReader(in) }

			top, err := Children(open(), 0, -1, -1, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(top.Nodes) != 1 || top.Nodes[0].Func != "{main}" || top.Nodes[0].Children != 1 || !top.Nodes[0].Done || !near(top.Nodes[0].Incl, 0.01) || top.Next != nil || top.Truncated {
				t.Fatalf("top = %+v", top)
			}

			kids, err := Children(open(), 0, off["App\\Kernel->handle"], -1, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := funcs(kids); len(got) != 3 || got[0] != "strlen" || got[1] != "include" || got[2] != "App\\Db->query" {
				t.Fatalf("children = %v", got)
			}
			p := kids.Parent
			if p == nil || !p.Done || p.Children != 3 || !near(p.Incl, 0.0098) || !near(p.Self, 0.0098-0.0001-0.002-0.0072) {
				t.Errorf("parent = %+v", p)
			}
			s := kids.Nodes[0]
			if s.Return == nil || *s.Return != "3" || len(s.Args) != 1 || s.Args[0] != "'abc'" || !near(s.Incl, 0.0001) {
				t.Errorf("strlen = %+v", s)
			}
			if inc := kids.Nodes[1]; inc.Include != "/app/src/routes.php" || inc.Children != 0 {
				t.Errorf("include = %+v", inc)
			}
			q := kids.Nodes[2]
			if q.Children != 1 || !near(q.Incl, 0.0072) || !near(q.Self, 0.0002) || q.Memory != 2000 {
				t.Errorf("Db->query = %+v", q)
			}

			// Paging: two per page, then the rest from Next.
			first, err := Children(open(), 0, off["App\\Kernel->handle"], -1, 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(first.Nodes) != 2 || first.Next == nil || *first.Next != off["App\\Db->query"] {
				t.Fatalf("first page = %v next %v", funcs(first), first.Next)
			}
			rest, err := Children(open(), 0, off["App\\Kernel->handle"], *first.Next, 2)
			if err != nil {
				t.Fatal(err)
			}
			if got := funcs(rest); len(got) != 1 || got[0] != "App\\Db->query" || rest.Next != nil || rest.Truncated {
				t.Errorf("second page = %v next %v", got, rest.Next)
			}
		})
	}
}

func TestChildren_BadOffset(t *testing.T) {
	off := entryOffsets(t)
	for _, o := range []int64{off["strlen"] + 1, off["App\\Kernel->handle"] - 1} {
		if _, err := Children(bytes.NewReader([]byte(sample)), 0, o, -1, 0); !errors.Is(err, ErrBadOffset) {
			t.Errorf("offset %d: err = %v, want ErrBadOffset", o, err)
		}
	}
}

func TestChildren_CutTrace(t *testing.T) {
	off := entryOffsets(t)
	cut := sample[:off["PDO->query"]] // ends inside Db->query
	page, err := Children(bytes.NewReader([]byte(cut)), 0, off["App\\Kernel->handle"], -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Truncated || len(page.Nodes) != 3 || page.Nodes[2].Done || page.Parent.Done {
		t.Errorf("cut page = %+v", page)
	}
}

func TestSearch(t *testing.T) {
	off := entryOffsets(t)
	in := gz(t, sample)

	all, err := Search(bytes.NewReader(in), 0, "QUERY", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(all); len(got) != 2 || got[0] != "App\\Db->query" || got[1] != "PDO->query" || all.Next != nil || all.Truncated {
		t.Fatalf("search = %v", got)
	}
	if p := all.Nodes[1].Path; len(p) != 3 || p[0].Func != "{main}" || p[1].Func != "App\\Kernel->handle" || p[2].Func != "App\\Db->query" || p[2].Offset != off["App\\Db->query"] {
		t.Errorf("PDO->query path = %+v", p)
	}
	if p := all.Nodes[0].Path; len(p) != 2 {
		t.Errorf("Db->query path = %+v", p)
	}
	if n := all.Nodes[0]; !n.Done || n.Children != 1 || !near(n.Incl, 0.0072) || !near(n.Self, 0.0002) {
		t.Errorf("Db->query = %+v", n)
	}

	one, err := Search(bytes.NewReader(in), 0, "query", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(one.Nodes) != 1 || one.Next == nil || *one.Next != off["PDO->query"] || !one.Nodes[0].Done {
		t.Fatalf("limit 1 = %+v", one)
	}
	next, err := Search(bytes.NewReader(in), 0, "query", *one.Next, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(next); len(got) != 1 || got[0] != "PDO->query" || next.Next != nil || len(next.Nodes[0].Path) != 3 {
		t.Errorf("next page = %v", got)
	}

	byFile, _ := Search(bytes.NewReader(in), 0, "db.php", 0, 0)
	if got := funcs(byFile); len(got) != 1 || got[0] != "PDO->query" {
		t.Errorf("by file = %v", got)
	}
}

func TestClip(t *testing.T) {
	long := string(bytes.Repeat([]byte("é"), maxValueLen))
	c := clip(long)
	if len(c) > maxValueLen+len("…") || !bytes.HasSuffix([]byte(c), []byte("…")) {
		t.Errorf("clip len %d", len(c))
	}
	for _, r := range c {
		if r == '�' {
			t.Fatal("clip split a rune")
		}
	}
}

// Xdebug writes the R line after the exit line of the call.
func TestReturnAfterExit(t *testing.T) {
	off := entryOffsets(t)
	s, err := Search(bytes.NewReader([]byte(sample)), 0, "strlen", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Nodes) != 1 || s.Nodes[0].Return == nil || *s.Nodes[0].Return != "3" {
		t.Errorf("search strlen = %+v", s.Nodes)
	}
	c, err := Children(bytes.NewReader([]byte(sample)), 0, off["strlen"], -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c.Parent == nil || c.Parent.Return == nil || *c.Parent.Return != "3" {
		t.Errorf("parent strlen = %+v", c.Parent)
	}

	// A full page waits for the R line of the match it closes last.
	tr := "Version: 3.4.5\nFile format: 4\nTRACE START [x]\n" +
		"1\t0\t0\t0.1\t1\t{main}\t1\t\t/a.php\t0\t0\n" +
		"2\t1\t0\t0.2\t1\tfa\t1\t\t/a.php\t2\t0\n" +
		"3\t2\t0\t0.3\t1\tfa_inner\t1\t\t/a.php\t3\t0\n" +
		"3\t2\t1\t0.4\t1\n" +
		"3\t2\tR\t\t\t'ri'\n" +
		"2\t1\t1\t0.5\t1\n" +
		"2\t1\tR\t\t\t'ra'\n" +
		"2\t3\t0\t0.6\t1\tfb\t1\t\t/a.php\t4\t0\n" +
		"2\t3\t1\t0.7\t1\n" +
		"1\t0\t1\t0.8\t1\nTRACE END   [x]\n"
	p, err := Search(bytes.NewReader([]byte(tr)), 0, "fa", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Nodes) != 1 || p.Nodes[0].Func != "fa" || p.Nodes[0].Return == nil || *p.Nodes[0].Return != "'ra'" || p.Next == nil || !p.Nodes[0].Done {
		t.Errorf("full page = %+v next %v", p.Nodes, p.Next)
	}
}

// midScript is a trace started by xdebug_start_trace() inside a function:
// the first calls are at level 3, then the untraced callers exit and a
// call at level 2 follows.
const midScript = "Version: 3.4.5\nFile format: 4\nTRACE START [x]\n" +
	"3\t1\t0\t0.1\t100\tone\t1\t\t/a.php\t10\t0\n" +
	"3\t1\t1\t0.2\t100\n" +
	"3\t2\t0\t0.3\t100\ttwo\t1\t\t/a.php\t11\t0\n" +
	"4\t3\t0\t0.4\t100\tstrlen\t0\t\t/a.php\t20\t1\t'ab'\n" +
	"4\t3\t1\t0.5\t100\n" +
	"4\t3\tR\t\t\t2\n" +
	"3\t2\t1\t0.6\t100\n" +
	"2\t0\t1\t0.7\t100\n" +
	"2\t4\t0\t0.8\t100\tafter\t1\t\t/a.php\t5\t0\n" +
	"2\t4\t1\t0.9\t100\n" +
	"1\t0\t1\t1.0\t100\n" +
	"\t\t\t1.1\t100\nTRACE END   [x]\n"

func TestMidScriptTrace(t *testing.T) {
	top, err := Children(bytes.NewReader([]byte(midScript)), 0, -1, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(top); len(got) != 3 || got[0] != "one" || got[1] != "two" || got[2] != "after" || top.Truncated {
		t.Fatalf("top = %v truncated=%v", got, top.Truncated)
	}
	if two := top.Nodes[1]; two.Children != 1 || !two.Done || !near(two.Incl, 0.3) || !near(two.Self, 0.2) {
		t.Errorf("two = %+v", two)
	}

	// Paging the top level keeps its meaning.
	first, _ := Children(bytes.NewReader([]byte(midScript)), 0, -1, -1, 1)
	if first.Next == nil {
		t.Fatal("no next")
	}
	rest, err := Children(bytes.NewReader([]byte(midScript)), 0, -1, *first.Next, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(rest); len(got) != 2 || got[0] != "two" || got[1] != "after" {
		t.Errorf("rest of top = %v", got)
	}

	kids, err := Children(bytes.NewReader([]byte(midScript)), 0, top.Nodes[1].Offset, -1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := funcs(kids); len(got) != 1 || got[0] != "strlen" || kids.Nodes[0].Return == nil || *kids.Nodes[0].Return != "2" {
		t.Errorf("callees of two = %+v", kids.Nodes)
	}

	s, err := Search(bytes.NewReader([]byte(midScript)), 0, "strlen", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Nodes) != 1 || len(s.Nodes[0].Path) != 1 || s.Nodes[0].Path[0].Func != "two" {
		t.Errorf("strlen path = %+v", s.Nodes)
	}
	a, _ := Search(bytes.NewReader([]byte(midScript)), 0, "after", 0, 0)
	if len(a.Nodes) != 1 || len(a.Nodes[0].Path) != 0 {
		t.Errorf("after path = %+v", a.Nodes)
	}
}

func TestChildren_FromMustBeACallee(t *testing.T) {
	off := entryOffsets(t)
	if _, err := Children(bytes.NewReader([]byte(sample)), 0, off["App\\Kernel->handle"], off["PDO->query"], 0); !errors.Is(err, ErrBadOffset) {
		t.Errorf("from deeper than a callee: err = %v", err)
	}
	if _, err := Children(bytes.NewReader([]byte(sample)), 0, off["App\\Db->query"], off["strlen"], 0); !errors.Is(err, ErrBadOffset) {
		t.Errorf("from before parent: err = %v", err)
	}
}
