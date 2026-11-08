// Package xdebugtrace reads Xdebug function traces (xdebug.mode=trace) in
// the computer-readable format (xdebug.trace_format=1), plain or gzipped.
//
// Format 1 (Xdebug src/tracing/trace_computerized.c, "File format: 4"):
//
//	Version: 3.4.5
//	File format: 4
//	TRACE START [2026-11-08 10:00:00.000001]
//	level  fn#  0  time  memory  function  user(1)/internal(0)  include  file  line  [argc  args...]   entry
//	level  fn#  1  time  memory                                                                        exit
//	level  fn#  R           value                                                                      return (collect_return)
//	level       A                    file  line  $var = value                                          assignment (collect_assignments)
//	            time  memory                                                                           footer
//	TRACE END   [2026-11-08 10:00:00.100000]
//
// Fields are tab-separated. Traces are large (10^5–10^6 lines per framework
// request), so everything here streams and keeps memory bounded.
package xdebugtrace

import (
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Kind of a trace record.
type Kind byte

const (
	Entry  Kind = 'E'
	Exit   Kind = 'X'
	Return Kind = 'R'
	Assign Kind = 'A'
)

// Record is one data line of a format-1 trace.
type Record struct {
	Kind    Kind
	Level   int
	Nr      int     // function number; 0 for assignments
	Time    float64 // seconds since the trace started (entry, exit)
	Memory  int64   // bytes (entry, exit)
	Func    string  // entry
	User    bool    // entry: user-defined function
	Include string  // entry: file of include/require/eval
	File    string  // entry, assignment
	Line    int     // entry, assignment
	Args    []string
	Value   string // return value, or the assignment text ("$a = 1")
}

// Header is what the trace says about itself.
type Header struct {
	Version    string  `json:"xdebug_version"`
	FileFormat int     `json:"file_format"`
	Start      string  `json:"start"`      // TRACE START timestamp
	End        string  `json:"end"`        // TRACE END timestamp; empty when the trace was cut
	EndTime    float64 `json:"end_time"`   // footer: seconds at the end of the script
	EndMemory  int64   `json:"end_memory"` // footer: memory at the end of the script
}

var (
	// ErrNotFormat1 means the file is a trace in another format (0 = human
	// readable, 2 = HTML, 3/4 = flame graph) or not a trace at all.
	ErrNotFormat1 = errors.New("not an Xdebug trace in the computer-readable format: set xdebug.trace_format=1")
	// ErrTooLarge is returned by a Reader whose limit was reached.
	ErrTooLarge = errors.New("trace is larger than the limit")
)

// maxLine caps one line; the rest of a longer line is dropped (its last
// argument is cut). Xdebug caps values by var_display_max_*, so real lines
// stay far below it.
const maxLine = 1 << 20

// Reader decodes records one by one.
type Reader struct {
	br      *bufio.Reader
	pos     int64 // uncompressed bytes consumed by the lines read so far
	recOff  int64 // offset of the line of the record Next returned last
	limit   int64
	Header  Header
	records int
	line    int
	// Cut is true when a line was longer than maxLine and was shortened.
	Cut bool
}

// NewReader opens a trace stream; gzip is detected by its magic bytes.
// limit caps the uncompressed size (0 = none): past it Next returns ErrTooLarge.
func NewReader(r io.Reader, limit int64) (*Reader, error) {
	br := bufio.NewReaderSize(r, 64<<10)
	if magic, err := br.Peek(2); err == nil && magic[0] == 0x1f && magic[1] == 0x8b {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("gzip: %w", err)
		}
		br = bufio.NewReaderSize(zr, 64<<10)
	}
	return &Reader{br: br, limit: limit}, nil
}

// Offset is the uncompressed byte offset where the line of the record Next
// returned last starts.
func (t *Reader) Offset() int64 { return t.recOff }

// SkipTo moves the reader forward to the uncompressed offset off, which
// must be the start of a record line (an Offset from an earlier pass). The
// header is not read again, so the next record is taken as valid.
func (t *Reader) SkipTo(off int64) error {
	if off < t.pos {
		return fmt.Errorf("offset %d is behind the reader (%d)", off, t.pos)
	}
	if t.limit > 0 && off > t.limit {
		return ErrTooLarge
	}
	n, err := t.br.Discard(int(off - t.pos))
	t.pos += int64(n)
	if errors.Is(err, io.EOF) {
		return ErrBadOffset // past the end
	}
	if err != nil {
		return err
	}
	if t.records == 0 {
		t.records = 1
	}
	return nil
}

// Next returns the next record, io.EOF at the end.
func (t *Reader) Next() (Record, error) {
	for {
		if t.limit > 0 && t.pos > t.limit {
			return Record{}, ErrTooLarge
		}
		start := t.pos
		line, err := t.readLine()
		if err != nil {
			if err == io.EOF && t.records == 0 && t.Header.FileFormat == 0 {
				return Record{}, ErrNotFormat1
			}
			return Record{}, err
		}
		t.line++
		if line == "" {
			continue
		}
		if t.header(line) {
			continue
		}
		rec, ok := parseRecord(line)
		if !ok {
			if t.records == 0 {
				return Record{}, ErrNotFormat1
			}
			continue // a stray line inside a valid trace: skip it
		}
		t.records++
		t.recOff = start
		return rec, nil
	}
}

func (t *Reader) readLine() (string, error) {
	var sb strings.Builder
	read := 0
	for {
		chunk, err := t.br.ReadSlice('\n')
		read += len(chunk)
		if sb.Len()+len(chunk) <= maxLine {
			sb.Write(chunk)
		} else {
			t.Cut = true
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		t.pos += int64(read)
		if err != nil && (err != io.EOF || read == 0) {
			return "", err
		}
		return strings.TrimRight(sb.String(), "\r\n"), nil
	}
}

// header reads the non-record lines; true when line was one.
func (t *Reader) header(line string) bool {
	switch {
	case strings.HasPrefix(line, "Version: "):
		t.Header.Version = strings.TrimSpace(line[len("Version: "):])
	case strings.HasPrefix(line, "File format: "):
		t.Header.FileFormat, _ = strconv.Atoi(strings.TrimSpace(line[len("File format: "):]))
	case strings.HasPrefix(line, "TRACE START"):
		t.Header.Start = bracket(line)
	case strings.HasPrefix(line, "TRACE END"):
		t.Header.End = bracket(line)
	case strings.HasPrefix(line, "\t\t\t"):
		// Footer: time and memory at the end of the script.
		f := strings.Split(line, "\t")
		if len(f) >= 5 {
			t.Header.EndTime, _ = strconv.ParseFloat(f[3], 64)
			t.Header.EndMemory, _ = strconv.ParseInt(f[4], 10, 64)
		}
	default:
		return false
	}
	return true
}

func bracket(s string) string {
	i, j := strings.IndexByte(s, '['), strings.LastIndexByte(s, ']')
	if i < 0 || j <= i {
		return ""
	}
	return s[i+1 : j]
}

func parseRecord(line string) (Record, bool) {
	f := strings.Split(line, "\t")
	if len(f) < 3 {
		return Record{}, false
	}
	var r Record
	var err error
	if r.Level, err = strconv.Atoi(f[0]); err != nil {
		return r, false
	}
	switch f[2] {
	case "0": // entry
		if len(f) < 10 {
			return r, false
		}
		r.Kind = Entry
		if r.Nr, err = strconv.Atoi(f[1]); err != nil {
			return r, false
		}
		if r.Time, err = strconv.ParseFloat(f[3], 64); err != nil {
			return r, false
		}
		r.Memory, _ = strconv.ParseInt(f[4], 10, 64)
		r.Func = f[5]
		r.User = f[6] == "1"
		r.Include = f[7]
		r.File = f[8]
		r.Line, _ = strconv.Atoi(f[9])
		if len(f) > 10 {
			n, _ := strconv.Atoi(f[10])
			args := f[11:]
			if n >= 0 && n < len(args) {
				args = args[:n]
			}
			r.Args = args
		}
	case "1": // exit
		if len(f) < 5 {
			return r, false
		}
		r.Kind = Exit
		if r.Nr, err = strconv.Atoi(f[1]); err != nil {
			return r, false
		}
		if r.Time, err = strconv.ParseFloat(f[3], 64); err != nil {
			return r, false
		}
		r.Memory, _ = strconv.ParseInt(f[4], 10, 64)
	case "R":
		r.Kind = Return
		if r.Nr, err = strconv.Atoi(f[1]); err != nil {
			return r, false
		}
		if len(f) >= 6 {
			r.Value = strings.Join(f[5:], "\t")
		}
	case "A":
		r.Kind = Assign
		// level, "", "A", 5 empty fields, file, line, "$var = value".
		if len(f) >= 11 {
			r.File = f[8]
			r.Line, _ = strconv.Atoi(f[9])
			r.Value = strings.Join(f[10:], "\t")
		}
	default:
		return r, false
	}
	return r, true
}
