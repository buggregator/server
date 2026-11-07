package xdebug

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Saved breakpoints: kept in SQLite (xdebug_breakpoints), sent to every
// session Buggregator drives (no IDE). An IDE sets its own breakpoints.

// Breakpoint types.
const (
	BPLine      = "line"
	BPException = "exception"
)

var (
	errBPNotFound  = errors.New("no such breakpoint")
	errBPCondition = errors.New("a breakpoint condition is PHP code: it needs xdebug.allow_eval")
)

// SavedBreakpoint is a breakpoint set from the UI or by an agent.
type SavedBreakpoint struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`                // line or exception
	File      string    `json:"file,omitempty"`      // file:// URI as PHP sees it (line)
	Line      int       `json:"line,omitempty"`      // line
	Exception string    `json:"exception,omitempty"` // class name, e.g. \Throwable (exception)
	Condition string    `json:"condition,omitempty"` // PHP expression; sent only with allow_eval
	Enabled   bool      `json:"enabled"`
	Source    string    `json:"source"` // ui or mcp
	CreatedAt time.Time `json:"created_at"`
}

// normalize checks a breakpoint and fills defaults.
func (b *SavedBreakpoint) normalize() error {
	if b.Type == "" {
		b.Type = BPLine
		if b.Exception != "" && b.File == "" {
			b.Type = BPException
		}
	}
	b.File = fileURI(strings.TrimSpace(b.File))
	b.Exception = strings.TrimSpace(b.Exception)
	switch b.Type {
	case BPLine:
		if b.File == "" || b.Line < 1 {
			return errors.New("a line breakpoint needs file and line (>= 1)")
		}
		if !strings.HasPrefix(b.File, "file://") {
			return errors.New("file must be an absolute path as PHP sees it (/var/www/x.php, C:\\app\\x.php) or a file:// URI")
		}
		b.Exception = ""
	case BPException:
		if b.Exception == "" {
			return errors.New("an exception breakpoint needs exception (a class name, e.g. \\Throwable)")
		}
		b.File, b.Line = "", 0
	default:
		return fmt.Errorf("type must be line or exception, got %q", b.Type)
	}
	if strings.ContainsRune(b.File, 0) || strings.ContainsRune(b.Exception, 0) || strings.ContainsAny(b.Exception, " \t\"'") {
		return errors.New("file and exception must not contain NUL; exception must be a class name")
	}
	if b.Source != "mcp" {
		b.Source = "ui"
	}
	return nil
}

// sig changes when a breakpoint must be sent again.
func (b SavedBreakpoint) sig() string {
	return b.Type + "\x00" + b.File + "\x00" + fmt.Sprint(b.Line) + "\x00" + b.Exception + "\x00" + b.Condition
}

// fileURI turns an absolute path into the file:// URI Xdebug uses:
// /app/x.php → file:///app/x.php, C:\app\x.php → file:///C:/app/x.php.
// Anything else is returned as is and refused by normalize.
func fileURI(f string) string {
	switch {
	case strings.HasPrefix(f, "/"):
		return "file://" + f
	case len(f) > 2 && f[1] == ':' && (f[2] == '\\' || f[2] == '/') && isLetter(f[0]):
		return "file:///" + f[:2] + strings.ReplaceAll(f[2:], "\\", "/")
	}
	return f
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// bpStore keeps the saved breakpoints, cached in memory; without a database
// (tests) only in memory.
type bpStore struct {
	mu     sync.Mutex
	db     *sql.DB
	list   []SavedBreakpoint
	nextID int64
}

func (st *bpStore) load(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, type, file, line, exception, condition, enabled, source, created_at FROM xdebug_breakpoints ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var list []SavedBreakpoint
	for rows.Next() {
		var b SavedBreakpoint
		var enabled int
		var created string
		if err := rows.Scan(&b.ID, &b.Type, &b.File, &b.Line, &b.Exception, &b.Condition, &enabled, &b.Source, &created); err != nil {
			return err
		}
		b.Enabled = enabled == 1
		b.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		list = append(list, b)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	st.mu.Lock()
	st.db, st.list = db, list
	st.mu.Unlock()
	return nil
}

// List returns the breakpoints, oldest first.
func (st *bpStore) List() []SavedBreakpoint {
	st.mu.Lock()
	defer st.mu.Unlock()
	return append([]SavedBreakpoint{}, st.list...)
}

func (st *bpStore) Add(b SavedBreakpoint) (SavedBreakpoint, error) {
	if err := b.normalize(); err != nil {
		return b, err
	}
	b.CreatedAt = time.Now().UTC()
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.db != nil {
		res, err := st.db.Exec(`INSERT INTO xdebug_breakpoints (type, file, line, exception, condition, enabled, source, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			b.Type, b.File, b.Line, b.Exception, b.Condition, boolInt(b.Enabled), b.Source, b.CreatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return b, err
		}
		if b.ID, err = res.LastInsertId(); err != nil {
			return b, err
		}
	} else {
		st.nextID++
		b.ID = st.nextID
	}
	st.list = append(st.list, b)
	return b, nil
}

// Update applies fn to a breakpoint and saves it.
func (st *bpStore) Update(id int64, fn func(*SavedBreakpoint) error) (SavedBreakpoint, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.list {
		if st.list[i].ID != id {
			continue
		}
		b := st.list[i]
		if err := fn(&b); err != nil {
			return b, err
		}
		if err := b.normalize(); err != nil {
			return b, err
		}
		if st.db != nil {
			if _, err := st.db.Exec(`UPDATE xdebug_breakpoints SET type = ?, file = ?, line = ?, exception = ?, condition = ?, enabled = ? WHERE id = ?`,
				b.Type, b.File, b.Line, b.Exception, b.Condition, boolInt(b.Enabled), b.ID); err != nil {
				return b, err
			}
		}
		st.list[i] = b
		return b, nil
	}
	return SavedBreakpoint{}, errBPNotFound
}

func (st *bpStore) Delete(id int64) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i := range st.list {
		if st.list[i].ID != id {
			continue
		}
		if st.db != nil {
			if _, err := st.db.Exec(`DELETE FROM xdebug_breakpoints WHERE id = ?`, id); err != nil {
				return err
			}
		}
		st.list = append(st.list[:i], st.list[i+1:]...)
		return nil
	}
	return errBPNotFound
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// --- Per session ---------------------------------------------------------------

// Breakpoint states in a session driven here.
const (
	BPResolved   = "resolved"   // Xdebug found a line to stop at (resolved_line)
	BPUnresolved = "unresolved" // set, not resolved yet (the file is not loaded)
	BPSkipped    = "skipped"    // not sent: it has a condition and eval is off
	BPDisabled   = "disabled"   // switched off
	BPPending    = "pending"    // saved while the script ran: sent at the next pause
)

// sessionBP is a saved breakpoint as sent to one session.
type sessionBP struct {
	engineID     string
	sig          string
	state        string
	resolvedLine int
	hitCount     int
}

// SessionBreakpoint is a saved breakpoint with its state in one session.
type SessionBreakpoint struct {
	SavedBreakpoint
	State        string `json:"state"`
	ResolvedLine int    `json:"resolved_line,omitempty"`
	HitCount     int    `json:"hit_count"`
}

// syncBreakpoints sends the saved breakpoints to a session driven here:
// set the new and changed ones, remove the deleted and disabled ones. Only
// at the start or while paused (the engine reads commands only then);
// otherwise it marks the session dirty and the next pause syncs again. It
// reads the current list after taking bpSyncMu, so a sync that waited for
// another one never applies an older list. A remove the engine did not take
// keeps the engine id, so a later sync still removes it.
func (s *Session) syncBreakpoints() {
	if s.bpSource == nil {
		return
	}
	s.bpSyncMu.Lock()
	defer s.bpSyncMu.Unlock()
	s.mu.Lock()
	if s.driver != DriverHere || s.closed {
		s.mu.Unlock()
		return
	}
	if !s.canSendLocked() {
		s.bpDirty = true
		s.mu.Unlock()
		return
	}
	s.bpDirty = false
	s.mu.Unlock()
	list, allowEval := s.bpSource()

	want := map[int64]bool{}
	for _, b := range list {
		want[b.ID] = true
		s.mu.Lock()
		cur := s.bps[b.ID]
		s.mu.Unlock()
		send := b.Enabled && (b.Condition == "" || allowEval)
		if cur != nil && cur.engineID != "" && (!send || cur.sig != b.sig()) {
			if err := s.ownCall("breakpoint_remove -d " + cur.engineID); err != nil && !isDBGpError(err) {
				// Not removed (the script moved on): keep the engine id; the
				// next pause tries again. Show it as pending meanwhile.
				s.mu.Lock()
				cur.state = BPPending
				s.mu.Unlock()
				continue
			}
			s.mu.Lock()
			cur.engineID = ""
			s.mu.Unlock()
		}
		state := ""
		switch {
		case !b.Enabled:
			state = BPDisabled
		case !send:
			state = BPSkipped
		}
		if state != "" {
			s.setBP(b.ID, &sessionBP{sig: b.sig(), state: state})
			continue
		}
		if cur != nil && cur.engineID != "" && cur.sig == b.sig() {
			continue
		}
		body, err := s.ownCallReply(breakpointSetCmd(b))
		if err != nil {
			st := BPPending
			if isDBGpError(err) {
				st = BPUnresolved // Xdebug refused it (e.g. bad file): nothing set
			}
			s.setBP(b.ID, &sessionBP{sig: b.sig(), state: st})
			continue
		}
		id, resolved := parseBreakpointSet(body)
		st := BPUnresolved
		if resolved == "resolved" || (resolved == "" && b.Type == BPException) {
			st = BPResolved
		}
		s.setBP(b.ID, &sessionBP{engineID: id, sig: b.sig(), state: st})
	}
	// Deleted breakpoints: forget one only after the engine removed it.
	s.mu.Lock()
	gone := map[int64]string{}
	for id, cur := range s.bps {
		if !want[id] {
			gone[id] = cur.engineID
		}
	}
	s.mu.Unlock()
	for id, eid := range gone {
		if eid != "" {
			if err := s.ownCall("breakpoint_remove -d " + eid); err != nil && !isDBGpError(err) {
				continue // dirty: retried at the next pause
			}
		}
		s.mu.Lock()
		delete(s.bps, id)
		s.mu.Unlock()
	}
}

func isDBGpError(err error) bool {
	var de *dbgpError
	return errors.As(err, &de)
}

// setBP records a breakpoint as sent. A breakpoint_resolved notify that came
// for its engine id before the breakpoint_set reply was handled (Xdebug sends
// it right with the reply for a loaded file) is applied now.
func (s *Session) setBP(id int64, b *sessionBP) {
	s.mu.Lock()
	if s.bps == nil {
		s.bps = map[int64]*sessionBP{}
	}
	s.bps[id] = b
	var line int
	early := false
	if b.engineID != "" {
		if r, ok := s.bpEarly[b.engineID]; ok {
			delete(s.bpEarly, b.engineID)
			early = true
			line = r
			b.state, b.resolvedLine = BPResolved, r
		}
	}
	s.mu.Unlock()
	if early {
		s.emit("breakpoint.resolved", map[string]any{"session_id": s.ID, "breakpoint_id": id, "line": line})
	}
}

// breakpointSetCmd builds breakpoint_set for a saved breakpoint.
func breakpointSetCmd(b SavedBreakpoint) string {
	if b.Type == BPException {
		return "breakpoint_set -t exception -x " + quoteArg(b.Exception)
	}
	if b.Condition != "" {
		return fmt.Sprintf("breakpoint_set -t conditional -f %s -n %d -- %s", quoteArg(b.File), b.Line, b64Arg(b.Condition))
	}
	return fmt.Sprintf("breakpoint_set -t line -f %s -n %d", quoteArg(b.File), b.Line)
}

// parseBreakpointSet reads id and resolved from a breakpoint_set reply.
func parseBreakpointSet(body []byte) (id, resolved string) {
	var r struct {
		ID       string `xml:"id,attr"`
		Resolved string `xml:"resolved,attr"`
	}
	_ = unmarshalXML(body, &r)
	return r.ID, r.Resolved
}

// onNotify handles engine notifications of a session driven here:
// breakpoint_resolved moves a breakpoint to its real line.
func (s *Session) onNotify(body []byte) {
	var n struct {
		Name string `xml:"name,attr"`
		BP   struct {
			ID       string `xml:"id,attr"`
			Resolved string `xml:"resolved,attr"`
			Lineno   int    `xml:"lineno,attr"`
		} `xml:"breakpoint"`
	}
	if err := unmarshalXML(body, &n); err != nil || n.Name != "breakpoint_resolved" || n.BP.ID == "" {
		return
	}
	resolved := n.BP.Resolved == "" || n.BP.Resolved == "resolved"
	s.mu.Lock()
	var savedID int64
	for id, b := range s.bps {
		if b.engineID == n.BP.ID {
			savedID = id
			if resolved {
				b.state = BPResolved
			}
			b.resolvedLine = n.BP.Lineno
		}
	}
	if savedID == 0 && resolved && s.driver == DriverHere {
		// Not matched yet: the breakpoint_set reply is still on its way to
		// the sync. Keep it for setBP (few entries: one per set breakpoint).
		if s.bpEarly == nil {
			s.bpEarly = map[string]int{}
		}
		if len(s.bpEarly) < 1000 {
			s.bpEarly[n.BP.ID] = n.BP.Lineno
		}
	}
	s.mu.Unlock()
	if savedID != 0 {
		s.emit("breakpoint.resolved", map[string]any{"session_id": s.ID, "breakpoint_id": savedID, "line": n.BP.Lineno})
	}
}

// SessionBreakpoints returns the saved breakpoints with their state in this
// session. While paused it reads hit counts and lines from the engine.
func (s *Session) SessionBreakpoints(list []SavedBreakpoint) []SessionBreakpoint {
	if s.Paused() {
		if r, err := s.readResponse("breakpoint_list", ""); err == nil {
			s.mu.Lock()
			for _, eb := range r.Breakpoints {
				for _, b := range s.bps {
					if b.engineID == eb.ID {
						b.hitCount = eb.HitCount
						if eb.Resolved == "resolved" && b.state == BPUnresolved {
							b.state = BPResolved
						}
						if eb.Lineno > 0 && b.state == BPResolved {
							b.resolvedLine = eb.Lineno
						}
					}
				}
			}
			s.mu.Unlock()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SessionBreakpoint, 0, len(list))
	for _, b := range list {
		sb := SessionBreakpoint{SavedBreakpoint: b, State: BPPending}
		if cur := s.bps[b.ID]; cur != nil {
			sb.State, sb.ResolvedLine, sb.HitCount = cur.state, cur.resolvedLine, cur.hitCount
			// Changed since the last sync (it waits for a pause): not applied yet.
			if cur.sig != b.sig() || (cur.state == BPDisabled) != !b.Enabled {
				sb.State = BPPending
			}
		}
		out = append(out, sb)
	}
	return out
}

// --- Files seen ----------------------------------------------------------------

const maxSeenFiles = 500

// seenFiles remembers file URIs from stacks and pauses, newest last, so the
// UI can offer them for a new breakpoint.
type seenFiles struct {
	mu    sync.Mutex
	order []string
	at    map[string]int
	tick  int
}

func (f *seenFiles) see(uri string) {
	if uri == "" || !strings.HasPrefix(uri, "file://") {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.at == nil {
		f.at = map[string]int{}
	}
	f.tick++
	if _, ok := f.at[uri]; !ok {
		f.order = append(f.order, uri)
	}
	f.at[uri] = f.tick
	if len(f.order) > maxSeenFiles {
		sort.Slice(f.order, func(i, j int) bool { return f.at[f.order[i]] > f.at[f.order[j]] })
		for _, u := range f.order[maxSeenFiles:] {
			delete(f.at, u)
		}
		f.order = f.order[:maxSeenFiles]
	}
}

// find returns up to limit URIs matching q (prefix of the base name or the
// path first, then substring), most recent first.
func (f *seenFiles) find(q string, limit int, extra []string) []string {
	f.mu.Lock()
	all := append([]string{}, f.order...)
	at := make(map[string]int, len(f.at))
	for k, v := range f.at {
		at[k] = v
	}
	f.mu.Unlock()
	for _, e := range extra {
		if _, ok := at[e]; !ok && e != "" {
			all = append(all, e)
			at[e] = 0
		}
	}
	q = strings.ToLower(q)
	rank := func(u string) int {
		l := strings.ToLower(u)
		switch {
		case q == "":
			return 1
		case strings.HasPrefix(strings.ToLower(baseName(u)), q), strings.HasPrefix(strings.TrimPrefix(l, "file://"), q):
			return 2
		case strings.Contains(l, q):
			return 1
		}
		return 0
	}
	var hits []string
	for _, u := range all {
		if rank(u) > 0 {
			hits = append(hits, u)
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		ri, rj := rank(hits[i]), rank(hits[j])
		if ri != rj {
			return ri > rj
		}
		return at[hits[i]] > at[hits[j]]
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}
