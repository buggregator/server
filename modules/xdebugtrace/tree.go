package xdebugtrace

import (
	"errors"
	"io"
	"strings"
)

// Caps on what one API answer carries.
const (
	MaxNodes    = 1000 // nodes per tree page or search page
	maxValueLen = 512  // one argument or return value, in bytes
	maxArgs     = 20   // arguments per call
)

// Node is one call as the tree and search APIs return it.
type Node struct {
	Nr       int      `json:"nr"`
	Func     string   `json:"func"`
	User     bool     `json:"user"`
	Include  string   `json:"include,omitempty"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Level    int      `json:"level"`
	Start    float64  `json:"start"`
	Incl     float64  `json:"incl"`
	Self     float64  `json:"self"`
	Memory   int64    `json:"memory"` // memory change from entry to exit
	Args     []string `json:"args,omitempty"`
	Return   *string  `json:"return,omitempty"`
	Children int      `json:"children"` // direct callees
	Done     bool     `json:"done"`     // its exit was seen (false when the trace was cut)
	Offset   int64    `json:"offset"`   // pass as parent= or from=
	// Path is set on search results: the callers from the top level down
	// to the direct caller.
	Path []PathItem `json:"path,omitempty"`

	mem       int64
	childTime float64
	childOpen map[int]float64
}

// PathItem is one caller on a search result's Path.
type PathItem struct {
	Nr     int    `json:"nr"`
	Func   string `json:"func"`
	Offset int64  `json:"offset"`
}

// Page is a slice of nodes and where the next one starts.
type Page struct {
	Parent    *Node  `json:"parent,omitempty"`
	Nodes     []Node `json:"nodes"`
	Next      *int64 `json:"next"`      // offset for from=, nil when there is no more
	Truncated bool   `json:"truncated"` // the trace ended or hit the size limit inside the scan
}

func newNode(r Record, off int64) *Node {
	n := &Node{
		Nr: r.Nr, Func: r.Func, User: r.User, Include: r.Include, File: r.File, Line: r.Line,
		Level: r.Level, Start: r.Time, Offset: off, mem: r.Memory,
	}
	args := r.Args
	if len(args) > maxArgs {
		args = args[:maxArgs]
	}
	for _, a := range args {
		n.Args = append(n.Args, clip(a))
	}
	return n
}

func clip(s string) string {
	if len(s) <= maxValueLen {
		return s
	}
	cut := maxValueLen
	for cut > 0 && cut < len(s) && s[cut]&0xC0 == 0x80 { // keep UTF-8 whole
		cut--
	}
	return s[:cut] + "…"
}

func (n *Node) exit(r Record) {
	n.Incl = max(r.Time-n.Start, 0)
	n.Self = max(n.Incl-n.childTime, 0)
	n.Memory = r.Memory - n.mem
	n.Done = true
	n.childOpen = nil
}

// Children lists one level of calls. parent < 0: the top-level calls — the
// calls that start while no listed call is open, so a trace started by
// xdebug_start_trace() deep in a script (first level 3, later 2) lists
// right. parent >= 0: the direct callees of the call whose entry line
// starts there. from >= 0 resumes the same list at a Page.Next (pass the
// same parent). limit caps the nodes (0 or more than MaxNodes = MaxNodes).
func Children(r io.Reader, sizeLimit, parent, from int64, limit int) (*Page, error) {
	if limit <= 0 || limit > MaxNodes {
		limit = MaxNodes
	}
	tr, err := NewReader(r, sizeLimit)
	if err != nil {
		return nil, err
	}
	page := &Page{Nodes: []Node{}}

	entryAt := func(off int64) (Record, error) {
		if err := tr.SkipTo(off); err != nil {
			return Record{}, err
		}
		rec, err := tr.Next()
		if err != nil || rec.Kind != Entry || tr.Offset() != off {
			return Record{}, ErrBadOffset
		}
		return rec, nil
	}
	if parent >= 0 {
		rec, err := entryAt(parent)
		if err != nil {
			return nil, err
		}
		page.Parent = newNode(rec, parent)
	}
	var first *Record
	if from >= 0 {
		rec, err := entryAt(from)
		if err != nil {
			return nil, err
		}
		if page.Parent != nil && rec.Level != page.Parent.Level+1 {
			return nil, ErrBadOffset // not a callee of parent
		}
		first = &rec
	}

	var cur *Node // the listed call open now, or the last one closed
	add := func(rec Record, off int64) bool {
		if len(page.Nodes) == limit {
			page.Next = &off
			return false
		}
		page.Nodes = append(page.Nodes, *newNode(rec, off))
		cur = &page.Nodes[len(page.Nodes)-1]
		cur.childOpen = map[int]float64{}
		return true
	}
	if first != nil {
		add(*first, from)
	}

	for {
		rec, err := tr.Next()
		if err != nil {
			if errors.Is(err, ErrTooLarge) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				// The end of a whole trace is no cut: only the size limit or
				// a call left without its exit is.
				page.Truncated = !errors.Is(err, io.EOF) || (page.Parent != nil && !page.Parent.Done) || (cur != nil && !cur.Done)
				break
			}
			return nil, err
		}
		if rec.Kind == Assign {
			continue
		}
		if p := page.Parent; p != nil && rec.Level <= p.Level {
			if rec.Kind == Exit && rec.Nr == p.Nr {
				p.exit(rec)
				// Xdebug writes the return value after the exit line.
				if next, err := tr.Next(); err == nil && next.Kind == Return && next.Nr == p.Nr {
					v := clip(next.Value)
					p.Return = &v
				}
			}
			break // the parent's exit ends its callees
		}
		open := cur != nil && !cur.Done
		switch {
		case rec.Kind == Entry && !open:
			if !add(rec, tr.Offset()) {
				return page, nil
			}
		case cur == nil:
			// an exit of a caller above the traced part
		case rec.Nr == cur.Nr && rec.Kind == Exit && open:
			cur.exit(rec)
		case rec.Nr == cur.Nr && rec.Kind == Return:
			v := clip(rec.Value)
			cur.Return = &v
		case open && rec.Level == cur.Level+1 && rec.Kind == Entry:
			cur.Children++
			cur.childOpen[rec.Nr] = rec.Time
		case open && rec.Level == cur.Level+1 && rec.Kind == Exit:
			if start, ok := cur.childOpen[rec.Nr]; ok {
				cur.childTime += max(rec.Time-start, 0)
				delete(cur.childOpen, rec.Nr)
			}
		}
	}
	if page.Parent != nil && page.Parent.Done && from < 0 && page.Next == nil {
		// The listed callees end with the parent, so it has exactly as many.
		page.Parent.Children = len(page.Nodes)
		var sum float64
		for _, n := range page.Nodes {
			sum += n.Incl
		}
		page.Parent.Self = max(page.Parent.Incl-sum, 0)
	}
	return page, nil
}

// ErrBadOffset is returned when parent or from does not point at a call entry.
var ErrBadOffset = errors.New("offset is not the start of a call")

// Search lists calls whose function name or file contains q (case does not
// matter), in trace order, from the entry at from (0 = the start). Each
// match gets its time once its exit is read; the scan runs on past the
// limit only until the matches already taken are closed. Each match carries
// Path, its callers from the top level down, so a viewer can open the tree
// to it; that is why the scan always starts at the beginning, also for from.
func Search(r io.Reader, sizeLimit int64, q string, from int64, limit int) (*Page, error) {
	if limit <= 0 || limit > MaxNodes {
		limit = MaxNodes
	}
	q = strings.ToLower(q)
	tr, err := NewReader(r, sizeLimit)
	if err != nil {
		return nil, err
	}
	page := &Page{Nodes: []Node{}}
	var callers []PathItem     // open calls, outermost first
	open := map[int]int{}      // nr → index in page.Nodes
	stack := map[int]float64{} // nr → start of open calls one level below a match
	childOf := map[int]int{}   // nr of a direct callee → nr of its open matched caller
	levelOf := map[int]int{}   // nr of an open match → its level
	// Xdebug writes a return value on the line after the exit: the match
	// closed last waits one record for it, and a full page ends one later.
	closedNr, closedAt, finish := -1, -1, false
	for {
		rec, err := tr.Next()
		if err == nil {
			if rec.Kind == Return && rec.Nr == closedNr {
				v := clip(rec.Value)
				page.Nodes[closedAt].Return = &v
			}
			closedNr = -1
			if finish {
				return page, nil
			}
		}
		if err != nil {
			if errors.Is(err, ErrTooLarge) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				page.Truncated = !errors.Is(err, io.EOF) || len(open) > 0
				break
			}
			return nil, err
		}
		switch rec.Kind {
		case Entry:
			self := PathItem{Nr: rec.Nr, Func: rec.Func, Offset: tr.Offset()}
			path := callers
			callers = append(callers, self)
			for nr, lvl := range levelOf {
				if rec.Level == lvl+1 {
					page.Nodes[open[nr]].Children++
					stack[rec.Nr] = rec.Time
					childOf[rec.Nr] = nr
				}
			}
			if page.Next != nil || tr.Offset() < from {
				continue
			}
			if !strings.Contains(strings.ToLower(rec.Func), q) && !strings.Contains(strings.ToLower(rec.File), q) {
				continue
			}
			if len(page.Nodes) == limit {
				off := tr.Offset()
				page.Next = &off
				if len(open) == 0 {
					return page, nil
				}
				continue
			}
			n := newNode(rec, tr.Offset())
			n.Path = append([]PathItem{}, path...)
			page.Nodes = append(page.Nodes, *n)
			open[rec.Nr] = len(page.Nodes) - 1
			levelOf[rec.Nr] = rec.Level
		case Exit:
			// Pop to the exited call; exits of callers above the traced
			// part match nothing and leave the stack as it is.
			for k := len(callers) - 1; k >= 0; k-- {
				if callers[k].Nr == rec.Nr {
					callers = callers[:k]
					break
				}
			}
			if p, ok := childOf[rec.Nr]; ok {
				if i, ok := open[p]; ok {
					page.Nodes[i].childTime += max(rec.Time-stack[rec.Nr], 0)
				}
				delete(childOf, rec.Nr)
				delete(stack, rec.Nr)
			}
			if i, ok := open[rec.Nr]; ok {
				page.Nodes[i].exit(rec)
				delete(open, rec.Nr)
				delete(levelOf, rec.Nr)
				closedNr, closedAt = rec.Nr, i
				finish = page.Next != nil && len(open) == 0
			}
		case Return:
			if i, ok := open[rec.Nr]; ok {
				v := clip(rec.Value)
				page.Nodes[i].Return = &v
			}
		}
	}
	return page, nil
}
