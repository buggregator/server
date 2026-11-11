package xdebugtrace

import (
	"container/heap"
	"errors"
	"io"
	"sort"
)

// TopN is how many functions and calls a summary keeps.
const TopN = 20

// IndexEvery is how many calls lie between two index points.
const IndexEvery = 1000

// Summary is what one streaming pass learns about a trace.
type Summary struct {
	Header
	Calls       int        `json:"calls"`      // function entries
	UserCalls   int        `json:"user_calls"` // of them user-defined
	MaxDepth    int        `json:"max_depth"`
	Duration    float64    `json:"duration"`    // seconds, last time seen
	PeakMemory  int64      `json:"peak_memory"` // highest memory seen on entry/exit
	Assignments int        `json:"assignments"`
	Returns     bool       `json:"returns"`         // return values were collected
	Truncated   bool       `json:"truncated"`       // stopped at the size limit, or the trace has no end
	Main        string     `json:"main"`            // script of {main}
	Top         []FuncStat `json:"top"`             // by self time
	Slowest     []Call     `json:"slowest"`         // by inclusive time
	Index       []Point    `json:"index,omitempty"` // every IndexEvery-th call
}

// FuncStat aggregates one function over the trace.
type FuncStat struct {
	Func  string  `json:"func"`
	User  bool    `json:"user"`
	Count int     `json:"count"`
	Self  float64 `json:"self"` // seconds spent in the function itself
	Incl  float64 `json:"incl"` // seconds including callees; recursion counts each level
}

// Call is one function call.
type Call struct {
	Nr     int     `json:"nr"`
	Func   string  `json:"func"`
	File   string  `json:"file"`
	Line   int     `json:"line"`
	Level  int     `json:"level"`
	Start  float64 `json:"start"`
	Incl   float64 `json:"incl"`
	Self   float64 `json:"self"`
	Memory int64   `json:"memory"` // memory change from entry to exit
	Offset int64   `json:"offset"` // byte offset of its entry line
}

// Point lets a reader start in the middle of a trace: the entry line of call
// Nr starts at byte Offset (uncompressed), at depth Level.
type Point struct {
	Nr     int   `json:"nr"`
	Offset int64 `json:"offset"`
	Level  int   `json:"level"`
}

type frame struct {
	Call
	user  bool
	mem   int64
	child float64
}

// Summarize reads a whole trace once. limit caps the uncompressed size
// (0 = none); a trace over it is summarized up to the limit with Truncated set.
func Summarize(r io.Reader, limit int64) (*Summary, error) {
	tr, err := NewReader(r, limit)
	if err != nil {
		return nil, err
	}
	s := &Summary{}
	funcs := map[string]*FuncStat{}
	slow := &callHeap{}
	var stack []*frame
	var last float64

	closeFrame := func(f *frame, end float64, mem int64) {
		f.Incl = end - f.Start
		if f.Incl < 0 {
			f.Incl = 0
		}
		f.Self = f.Incl - f.child
		if f.Self < 0 {
			f.Self = 0
		}
		f.Memory = mem - f.mem
		fs := funcs[f.Func]
		if fs == nil {
			fs = &FuncStat{Func: f.Func, User: f.user}
			funcs[f.Func] = fs
		}
		fs.Count++
		fs.Self += f.Self
		fs.Incl += f.Incl
		if len(stack) > 0 {
			stack[len(stack)-1].child += f.Incl
		}
		pushTop(slow, f.Call)
	}

	for {
		rec, err := tr.Next()
		offset := tr.Offset()
		if errors.Is(err, io.EOF) {
			break
		}
		if errors.Is(err, ErrTooLarge) {
			s.Truncated = true
			break
		}
		if err != nil {
			if s.Calls == 0 {
				return nil, err
			}
			s.Truncated = true // a broken gzip tail: keep what was read
			break
		}
		if rec.Memory > s.PeakMemory {
			s.PeakMemory = rec.Memory
		}
		switch rec.Kind {
		case Entry:
			last = rec.Time
			if s.Calls%IndexEvery == 0 {
				s.Index = append(s.Index, Point{Nr: rec.Nr, Offset: offset, Level: rec.Level})
			}
			s.Calls++
			if rec.User {
				s.UserCalls++
			}
			if rec.Level > s.MaxDepth {
				s.MaxDepth = rec.Level
			}
			if rec.Func == "{main}" && s.Main == "" {
				s.Main = rec.File
			}
			stack = append(stack, &frame{
				Call: Call{Nr: rec.Nr, Func: rec.Func, File: rec.File, Line: rec.Line, Level: rec.Level, Start: rec.Time, Offset: offset},
				user: rec.User, mem: rec.Memory,
			})
		case Exit:
			last = rec.Time
			// Pop to the matching call; frames left open above it (an
			// exception unwinding) close at the same time.
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].Nr == rec.Nr {
					for len(stack) > i {
						f := stack[len(stack)-1]
						stack = stack[:len(stack)-1]
						closeFrame(f, rec.Time, rec.Memory)
					}
					break
				}
			}
		case Return:
			s.Returns = true
		case Assign:
			s.Assignments++
		}
	}

	s.Header = tr.Header
	if s.Header.EndTime > last {
		last = s.Header.EndTime
	}
	s.Duration = last
	if s.Header.EndMemory > s.PeakMemory {
		s.PeakMemory = s.Header.EndMemory
	}
	if len(stack) > 0 && s.Header.End == "" {
		s.Truncated = true
	}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		closeFrame(f, last, f.mem)
	}
	if s.Calls == 0 && s.Header.FileFormat == 0 {
		return nil, ErrNotFormat1
	}

	for _, fs := range funcs {
		s.Top = append(s.Top, *fs)
	}
	sort.Slice(s.Top, func(i, j int) bool {
		if s.Top[i].Self != s.Top[j].Self {
			return s.Top[i].Self > s.Top[j].Self
		}
		return s.Top[i].Func < s.Top[j].Func
	})
	if len(s.Top) > TopN {
		s.Top = s.Top[:TopN]
	}
	s.Slowest = make([]Call, slow.Len())
	for i := len(s.Slowest) - 1; i >= 0; i-- {
		s.Slowest[i] = heap.Pop(slow).(Call)
	}
	return s, nil
}

// callHeap is a min-heap by inclusive time holding the TopN slowest calls.
type callHeap []Call

func (h callHeap) Len() int { return len(h) }
func (h callHeap) Less(i, j int) bool {
	if h[i].Incl != h[j].Incl {
		return h[i].Incl < h[j].Incl
	}
	return h[i].Nr > h[j].Nr
}
func (h callHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *callHeap) Push(x any)   { *h = append(*h, x.(Call)) }
func (h *callHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}

func pushTop(h *callHeap, c Call) {
	if h.Len() < TopN {
		heap.Push(h, c)
		return
	}
	if (*h)[0].Incl < c.Incl {
		(*h)[0] = c
		heap.Fix(h, 0)
	}
}
