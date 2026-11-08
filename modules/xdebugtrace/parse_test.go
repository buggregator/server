package xdebugtrace

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// A trace as Xdebug 3.4 writes it with trace_format=1, collect_return=1 and
// collect_assignments=1 (layout from src/tracing/trace_computerized.c).
// Recorded traces: recorded_test.go.
const sample = "Version: 3.4.5\n" +
	"File format: 4\n" +
	"TRACE START [2026-11-08 10:00:00.000001]\n" +
	"1\t0\t0\t0.000100\t400000\t{main}\t1\t\t/app/index.php\t0\t0\n" +
	"2\t1\t0\t0.000200\t401000\tApp\\Kernel->handle\t1\t\t/app/index.php\t5\t1\tclass App\\Request { public $uri = '/users' }\n" +
	"3\t2\t0\t0.000300\t402000\tstrlen\t0\t\t/app/src/Kernel.php\t10\t1\t'abc'\n" +
	"3\t2\t1\t0.000400\t402000\n" +
	"3\t2\tR\t\t\t3\n" +
	"3\t\tA\t\t\t\t\t\t/app/src/Kernel.php\t11\t$n = 3\n" +
	"3\t3\t0\t0.000500\t403000\tinclude\t1\t/app/src/routes.php\t/app/src/Kernel.php\t12\t0\n" +
	"3\t3\t1\t0.002500\t450000\n" +
	"3\t4\t0\t0.002600\t450000\tApp\\Db->query\t1\t\t/app/src/Kernel.php\t13\t2\t'SELECT 1'\t[]\n" +
	"4\t5\t0\t0.002700\t451000\tPDO->query\t0\t\t/app/src/Db.php\t20\t1\t'SELECT 1'\n" +
	"4\t5\t1\t0.009700\t452000\n" +
	"3\t4\t1\t0.009800\t452000\n" +
	"2\t1\t1\t0.010000\t430000\n" +
	"1\t0\t1\t0.010100\t420000\n" +
	"\t\t\t0.010200\t419000\n" +
	"TRACE END   [2026-11-08 10:00:00.010201]\n\n"

func gz(t *testing.T, s string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	w.Write([]byte(s))
	w.Close()
	return b.Bytes()
}

func TestReader_Records(t *testing.T) {
	for name, in := range map[string][]byte{"plain": []byte(sample), "gzip": gz(t, sample), "crlf": []byte(strings.ReplaceAll(sample, "\n", "\r\n"))} {
		t.Run(name, func(t *testing.T) {
			r, err := NewReader(bytes.NewReader(in), 0)
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
				recs = append(recs, rec)
			}
			if len(recs) != 14 {
				t.Fatalf("records: %d", len(recs))
			}
			h := r.Header
			if h.Version != "3.4.5" || h.FileFormat != 4 || h.Start != "2026-11-08 10:00:00.000001" || h.End != "2026-11-08 10:00:00.010201" || h.EndTime != 0.0102 || h.EndMemory != 419000 {
				t.Errorf("header: %+v", h)
			}
			k := recs[1]
			if k.Kind != Entry || k.Level != 2 || k.Nr != 1 || k.Func != `App\Kernel->handle` || !k.User || k.File != "/app/index.php" || k.Line != 5 || len(k.Args) != 1 || k.Args[0] != `class App\Request { public $uri = '/users' }` {
				t.Errorf("entry: %+v", k)
			}
			if s := recs[2]; s.User || s.Func != "strlen" || s.Args[0] != "'abc'" {
				t.Errorf("internal: %+v", s)
			}
			if x := recs[3]; x.Kind != Exit || x.Nr != 2 || x.Time != 0.0004 || x.Memory != 402000 {
				t.Errorf("exit: %+v", x)
			}
			if rv := recs[4]; rv.Kind != Return || rv.Nr != 2 || rv.Value != "3" {
				t.Errorf("return: %+v", rv)
			}
			if a := recs[5]; a.Kind != Assign || a.File != "/app/src/Kernel.php" || a.Line != 11 || a.Value != "$n = 3" {
				t.Errorf("assign: %+v", a)
			}
			if inc := recs[6]; inc.Func != "include" || inc.Include != "/app/src/routes.php" || len(inc.Args) != 0 {
				t.Errorf("include: %+v", inc)
			}
			if q := recs[8]; len(q.Args) != 2 || q.Args[1] != "[]" {
				t.Errorf("args: %+v", q)
			}
		})
	}
}

func TestReader_OffsetPointsAtLine(t *testing.T) {
	r, _ := NewReader(strings.NewReader(sample), 0)
	for {
		rec, err := r.Next()
		if err == io.EOF {
			break
		}
		if rec.Kind != Entry {
			continue
		}
		off := r.Offset()
		line := sample[off:]
		if !strings.HasPrefix(line, fmt.Sprintf("%d\t%d\t0\t", rec.Level, rec.Nr)) {
			t.Errorf("nr %d: offset %d points at %q", rec.Nr, off, line[:min(30, len(line))])
		}
	}
}

func TestReader_NotFormat1(t *testing.T) {
	for name, in := range map[string]string{
		"format 0": "TRACE START [2026-11-08 10:00:00.000001]\n    0.0001     393168   -> {main}() /app/index.php:0\n",
		"html":     "<table class='xdebug-trace' dir='ltr' border='1' cellspacing='0'>\n<tr><th>#</th></tr>\n",
		"flame":    "{main} 3021\n{main};strlen 12\n",
		"empty":    "",
		"json":     `{"not":"a trace"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Summarize(strings.NewReader(in), 0); !errors.Is(err, ErrNotFormat1) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestReader_LongLineCut(t *testing.T) {
	long := strings.Repeat("x", maxLine+10)
	in := "File format: 4\n1\t0\t0\t0.1\t1\t{main}\t1\t\t/a.php\t0\t1\t'" + long + "'\n1\t0\t1\t0.2\t1\n"
	r, _ := NewReader(strings.NewReader(in), 0)
	rec, err := r.Next()
	if err != nil || rec.Func != "{main}" || !r.Cut {
		t.Fatalf("%v %+v cut=%v", err, rec.Func, r.Cut)
	}
	if x, err := r.Next(); err != nil || x.Kind != Exit {
		t.Fatalf("after a long line: %v %+v", err, x)
	}
}
