package codegraph

import (
	"database/sql"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// SymbolRef is a symbol as lists show it.
type SymbolRef struct {
	ID        int64  `json:"id"`
	Root      string `json:"root"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	FQN       string `json:"fqn"`
	File      string `json:"file"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Signature string `json:"signature,omitempty"`
	ParentID  int64  `json:"parent_id,omitempty"`
}

// Relation is a non-call link of a symbol: extends, implements, uses, imports.
type Relation struct {
	Kind   string     `json:"kind"`
	FQN    string     `json:"fqn"`
	Symbol *SymbolRef `json:"symbol,omitempty"` // nil: outside the indexed code
	Line   int        `json:"line"`
}

// SymbolDetail is one symbol with its members and links.
type SymbolDetail struct {
	SymbolRef
	Parent     *SymbolRef  `json:"parent,omitempty"`
	Members    []SymbolRef `json:"members"`
	Relations  []Relation  `json:"relations"`  // what this symbol extends, implements, uses, imports
	Incoming   []Relation  `json:"incoming"`   // who extends, implements, uses this symbol
	Callers    int         `json:"callers"`    // exact incoming calls
	NameCalls  int         `json:"name_calls"` // calls matched by method name only
	Callees    int         `json:"callees"`
	Source     *string     `json:"source,omitempty"` // only with code_graph.snippets on
	SourceFrom int         `json:"source_from,omitempty"`
}

// GraphNode is a node of a graph answer: a symbol, a rolled-up namespace or
// package, or a target outside the indexed code (External).
type GraphNode struct {
	ID       string `json:"id"` // "s:<symbol id>", "ns:<fqn>", "x:<fqn>", "f:<file>"
	SymbolID int64  `json:"symbol_id,omitempty"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
	FQN      string `json:"fqn"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	External bool   `json:"external,omitempty"`
	Depth    int    `json:"depth"`
	// Size is the number of types, functions and methods folded into the
	// node (graph answers only).
	Size int `json:"size,omitempty"`
	// Group is the node this one was expanded from (expand=).
	Group string `json:"group,omitempty"`
}

// GraphEdge links two nodes. Count is the number of references it stands for.
type GraphEdge struct {
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	Confidence string `json:"confidence"`
	Line       int    `json:"line,omitempty"`
	Count      int    `json:"count"`
}

// Graph is a set of nodes and edges, cut at a size limit.
type Graph struct {
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	Truncated bool        `json:"truncated"`
}

var errNotFound = errors.New("not found")

const symbolCols = `s.id, s.root, s.kind, s.name, s.fqn, f.path, s.line_start, s.line_end, s.signature, COALESCE(s.parent_id, 0)`

func scanSymbol(sc interface{ Scan(...any) error }) (SymbolRef, error) {
	var s SymbolRef
	err := sc.Scan(&s.ID, &s.Root, &s.Kind, &s.Name, &s.FQN, &s.File, &s.LineStart, &s.LineEnd, &s.Signature, &s.ParentID)
	return s, err
}

func querySymbols(db *sql.DB, q string, args ...any) ([]SymbolRef, error) {
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SymbolRef{}
	for rows.Next() {
		s, err := scanSymbol(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Graphs answers read queries over the stored graph.
type Graphs struct {
	db       *sql.DB
	ix       *Indexer
	snippets bool
	pathMap  []PathMap

	mu    sync.Mutex
	cache map[string]*symbolIndex // root -> symbols, dropped when indexed_at changes
}

func clampInt(v, def, lo, hi int) int {
	if v <= 0 {
		v = def
	}
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}

// Search finds symbols by name or full name. Namespaces and packages appear
// once per name, not once per file.
// forget drops the cached symbols of a root (removed; a root added again
// under the same name starts its run counter at 0).
func (g *Graphs) forget(root string) {
	g.mu.Lock()
	delete(g.cache, root)
	g.mu.Unlock()
}

func (g *Graphs) Search(q, kind, root string, limit int) ([]SymbolRef, error) {
	limit = clampInt(limit, 50, 1, 500)
	q = strings.TrimSpace(q)
	if q == "" {
		return []SymbolRef{}, nil
	}
	where := []string{`(s.name LIKE ? ESCAPE '\' OR s.fqn LIKE ? ESCAPE '\')`}
	esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	args := []any{esc + "%", "%" + esc + "%"}
	if kind != "" {
		kinds := strings.Split(kind, ",")
		where = append(where, "s.kind IN (?"+strings.Repeat(",?", len(kinds)-1)+")")
		for _, k := range kinds {
			args = append(args, strings.TrimSpace(k))
		}
	}
	if root != "" {
		where = append(where, "s.root = ?")
		args = append(args, root)
	}
	args = append(args, q, q, esc+"%", limit*4)
	rows, err := querySymbols(g.db, `SELECT `+symbolCols+` FROM code_symbols s JOIN code_files f ON f.id = s.file_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY (s.name = ? COLLATE NOCASE) DESC, (s.fqn = ? COLLATE NOCASE) DESC, (s.name LIKE ? ESCAPE '\') DESC, length(s.fqn), s.fqn
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	out := []SymbolRef{}
	seen := map[string]bool{}
	for _, s := range rows {
		if s.Kind == KindNamespace || s.Kind == KindPackage {
			key := s.Root + "\x00" + s.Kind + "\x00" + s.FQN
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		out = append(out, s)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// SymbolByID returns one symbol.
func (g *Graphs) SymbolByID(id int64) (SymbolRef, error) {
	s, err := scanSymbol(g.db.QueryRow(`SELECT `+symbolCols+` FROM code_symbols s JOIN code_files f ON f.id = s.file_id WHERE s.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return s, errNotFound
	}
	return s, err
}

// Lookup finds a symbol by id ("123") or by full name, optionally in a root.
func (g *Graphs) Lookup(ref, root string) (SymbolRef, error) {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return g.SymbolByID(id)
	}
	args := []any{strings.TrimPrefix(ref, `\`)}
	q := `SELECT ` + symbolCols + ` FROM code_symbols s JOIN code_files f ON f.id = s.file_id WHERE s.fqn = ?`
	if root != "" {
		q += ` AND s.root = ?`
		args = append(args, root)
	}
	s, err := scanSymbol(g.db.QueryRow(q+` ORDER BY s.id LIMIT 1`, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return s, errNotFound
	}
	return s, err
}

// Detail returns a symbol with members, relations and call counts.
func (g *Graphs) Detail(id int64) (*SymbolDetail, error) {
	s, err := g.SymbolByID(id)
	if err != nil {
		return nil, err
	}
	d := &SymbolDetail{SymbolRef: s, Relations: []Relation{}, Incoming: []Relation{}}
	if s.ParentID != 0 {
		if p, err := g.SymbolByID(s.ParentID); err == nil {
			d.Parent = &p
		}
	}
	if d.Members, err = querySymbols(g.db, `SELECT `+symbolCols+` FROM code_symbols s JOIN code_files f ON f.id = s.file_id
		WHERE s.parent_id = ? ORDER BY f.path, s.line_start`, id); err != nil {
		return nil, err
	}

	rels := func(q string, args ...any) ([]Relation, error) {
		rows, err := g.db.Query(q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []Relation
		var ids []int64
		for rows.Next() {
			var r Relation
			var other int64
			if err := rows.Scan(&r.Kind, &r.FQN, &other, &r.Line); err != nil {
				return nil, err
			}
			out = append(out, r)
			ids = append(ids, other)
		}
		rows.Close()
		for i, oid := range ids {
			if oid != 0 {
				if ref, err := g.SymbolByID(oid); err == nil {
					out[i].Symbol = &ref
				}
			}
		}
		return out, nil
	}
	nonCall := `('extends', 'implements', 'uses', 'imports')`
	if d.Relations, err = rels(`SELECT kind, to_fqn, COALESCE(to_id, 0), line FROM code_edges WHERE from_id = ? AND kind IN `+nonCall+` ORDER BY line`, id); err != nil {
		return nil, err
	}
	if d.Incoming, err = rels(`SELECT e.kind, COALESCE(s.fqn, ''), COALESCE(e.from_id, 0), e.line FROM code_edges e LEFT JOIN code_symbols s ON s.id = e.from_id
		WHERE e.to_id = ? AND e.kind IN ('extends', 'implements', 'uses') ORDER BY s.fqn LIMIT 500`, id); err != nil {
		return nil, err
	}
	if d.Relations == nil {
		d.Relations = []Relation{}
	}
	if d.Incoming == nil {
		d.Incoming = []Relation{}
	}
	_ = g.db.QueryRow(`SELECT COUNT(*) FROM code_edges WHERE to_id = ? AND kind IN ('calls', 'instantiates')`, id).Scan(&d.Callers)
	if s.Kind == KindMethod {
		_ = g.db.QueryRow(`SELECT COUNT(*) FROM code_edges WHERE root = ? AND to_name = ? AND confidence = 'name' AND kind = 'calls'`, s.Root, s.Name).Scan(&d.NameCalls)
	}
	_ = g.db.QueryRow(`SELECT COUNT(*) FROM code_edges WHERE from_id = ? AND kind IN ('calls', 'instantiates')`, id).Scan(&d.Callees)

	if g.snippets && s.Kind != KindNamespace && s.Kind != KindPackage {
		if root, ok := g.ix.root(s.Root); ok {
			to := s.LineEnd
			if to-s.LineStart > 400 {
				to = s.LineStart + 400
			}
			if src, err := readSource(root, s.File, s.LineStart, to); err == nil {
				d.Source, d.SourceFrom = &src, s.LineStart
			}
		}
	}
	return d, nil
}

func symNode(s SymbolRef, depth int) GraphNode {
	return GraphNode{ID: "s:" + strconv.FormatInt(s.ID, 10), SymbolID: s.ID, Kind: s.Kind, Name: s.Name, FQN: s.FQN, File: s.File, Line: s.LineStart, Depth: depth}
}

type callRow struct {
	other      int64 // the symbol on the other side; 0 = none
	fileID     int64
	kind       string
	toName     string
	toFQN      string
	line       int
	confidence string
}

// Calls walks call and instantiation edges from a symbol: callers (incoming)
// or callees (outgoing), up to depth levels. Callers of a method also include
// calls matched by its name only (confidence "name"). Callees that are not in
// the index become external nodes.
func (g *Graphs) Calls(id int64, incoming bool, depth, limit int) (*Graph, error) {
	depth = clampInt(depth, 1, 1, 5)
	limit = clampInt(limit, 200, 1, 2000)
	start, err := g.SymbolByID(id)
	if err != nil {
		return nil, err
	}
	gr := &Graph{Nodes: []GraphNode{symNode(start, 0)}, Edges: []GraphEdge{}}
	nodes := map[string]bool{gr.Nodes[0].ID: true}
	edges := map[string]int{}
	frontier := []SymbolRef{start}
	for level := 1; level <= depth && len(frontier) > 0; level++ {
		var next []SymbolRef
		for _, s := range frontier {
			rows, err := g.callRows(s, incoming)
			if err != nil {
				return nil, err
			}
			sid := "s:" + strconv.FormatInt(s.ID, 10)
			for _, r := range rows {
				var n GraphNode
				switch {
				case r.other != 0:
					o, err := g.SymbolByID(r.other)
					if err != nil {
						continue
					}
					n = symNode(o, level)
					if !nodes[n.ID] {
						next = append(next, o)
					}
				case incoming:
					var p string
					_ = g.db.QueryRow(`SELECT path FROM code_files WHERE id = ?`, r.fileID).Scan(&p)
					n = GraphNode{ID: "f:" + p, Kind: "file", Name: path.Base(p), FQN: p, File: p, Line: r.line, Depth: level}
				default:
					fqn := r.toFQN
					if fqn == "" {
						fqn = r.toName
					}
					n = GraphNode{ID: "x:" + fqn, Kind: "external", Name: r.toName, FQN: fqn, External: true, Depth: level}
				}
				if !nodes[n.ID] {
					if len(gr.Nodes) >= limit {
						gr.Truncated = true
						continue
					}
					nodes[n.ID] = true
					gr.Nodes = append(gr.Nodes, n)
				}
				from, to := sid, n.ID
				if incoming {
					from, to = n.ID, sid
				}
				key := from + "|" + to + "|" + r.kind + "|" + r.confidence
				if i, ok := edges[key]; ok {
					gr.Edges[i].Count++
					continue
				}
				edges[key] = len(gr.Edges)
				gr.Edges = append(gr.Edges, GraphEdge{From: from, To: to, Kind: r.kind, Confidence: r.confidence, Line: r.line, Count: 1})
			}
		}
		frontier = next
	}
	return gr, nil
}

func (g *Graphs) callRows(s SymbolRef, incoming bool) ([]callRow, error) {
	var rows *sql.Rows
	var err error
	if incoming {
		q := `SELECT COALESCE(from_id, 0), file_id, kind, to_name, to_fqn, line, confidence FROM code_edges
			WHERE to_id = ? AND kind IN ('calls', 'instantiates')`
		args := []any{s.ID}
		if s.Kind == KindMethod {
			q += ` UNION ALL SELECT COALESCE(from_id, 0), file_id, kind, to_name, to_fqn, line, confidence FROM code_edges
				WHERE root = ? AND to_name = ? AND confidence = 'name' AND kind = 'calls'`
			args = append(args, s.Root, s.Name)
		}
		rows, err = g.db.Query(q+` LIMIT 5000`, args...)
	} else {
		rows, err = g.db.Query(`SELECT COALESCE(to_id, 0), file_id, kind, to_name, to_fqn, line, confidence FROM code_edges
			WHERE from_id = ? AND kind IN ('calls', 'instantiates') ORDER BY line LIMIT 5000`, s.ID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []callRow
	for rows.Next() {
		var r callRow
		if err := rows.Scan(&r.other, &r.fileID, &r.kind, &r.toName, &r.toFQN, &r.line, &r.confidence); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TopSymbol is a function or method with its exact incoming call count.
type TopSymbol struct {
	SymbolRef
	Callers int `json:"callers"`
}

// testPath is an SQL condition: the file is a test (Go *_test.go, PHP
// *Test.php, or under a test/ or tests/ directory).
func testPath(col string) string {
	return `(` + col + ` GLOB '*_test.go' OR ` + col + ` GLOB '*Test.php' OR lower('/' || ` + col + `) GLOB '*/test/*' OR lower('/' || ` + col + `) GLOB '*/tests/*')`
}

// Top lists the functions and methods of a root called most often (exact
// calls only), leaving out tests and calls made from tests.
func (g *Graphs) Top(root string, limit int) ([]TopSymbol, error) {
	limit = clampInt(limit, 20, 1, 200)
	rows, err := g.db.Query(`SELECT `+symbolCols+`, COUNT(*) AS callers
		FROM code_edges e
		JOIN code_symbols s ON s.id = e.to_id
		JOIN code_files f ON f.id = s.file_id
		JOIN code_files cf ON cf.id = e.file_id
		WHERE e.root = ? AND e.kind = 'calls' AND s.kind IN ('function', 'method')
			AND NOT `+testPath("f.path")+` AND NOT `+testPath("cf.path")+`
		GROUP BY s.id
		ORDER BY callers DESC, s.fqn
		LIMIT ?`, root, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TopSymbol{}
	for rows.Next() {
		var t TopSymbol
		if err := rows.Scan(&t.ID, &t.Root, &t.Kind, &t.Name, &t.FQN, &t.File, &t.LineStart, &t.LineEnd, &t.Signature, &t.ParentID, &t.Callers); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// FileEntry is a directory or file in a root's tree.
type FileEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Dir        bool   `json:"dir"`
	Files      int    `json:"files,omitempty"` // directories: indexed files below
	Lang       string `json:"lang,omitempty"`
	Symbols    int    `json:"symbols"`
	ParseError bool   `json:"parse_error,omitempty"`
}

// Files lists the entries directly under dir ("" = the root) of a root.
func (g *Graphs) Files(root, dir string) ([]FileEntry, error) {
	dir = strings.Trim(path.Clean("/"+filepath.ToSlash(dir)), "/")
	prefix := ""
	if dir != "" {
		prefix = dir + "/"
	}
	rows, err := g.db.Query(`SELECT f.path, f.lang, f.parse_error, (SELECT COUNT(*) FROM code_symbols s WHERE s.file_id = f.id)
		FROM code_files f WHERE f.root = ? AND substr(f.path, 1, ?) = ? ORDER BY f.path`, root, len(prefix), prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	dirs := map[string]*FileEntry{}
	out := []FileEntry{}
	var order []string
	for rows.Next() {
		var p, lang string
		var perr, syms int
		if err := rows.Scan(&p, &lang, &perr, &syms); err != nil {
			return nil, err
		}
		rest := strings.TrimPrefix(p, prefix)
		if name, _, sub := strings.Cut(rest, "/"); sub {
			d, ok := dirs[name]
			if !ok {
				d = &FileEntry{Name: name, Path: prefix + name, Dir: true}
				dirs[name] = d
				order = append(order, name)
			}
			d.Files++
			d.Symbols += syms
			continue
		}
		out = append(out, FileEntry{Name: rest, Path: p, Lang: lang, Symbols: syms, ParseError: perr == 1})
	}
	entries := make([]FileEntry, 0, len(order)+len(out))
	sort.Strings(order)
	for _, n := range order {
		entries = append(entries, *dirs[n])
	}
	return append(entries, out...), rows.Err()
}

// OutlineItem is a symbol of a file with its members.
type OutlineItem struct {
	SymbolRef
	Members []*OutlineItem `json:"members"`
}

// Outline returns the symbols of one file as a tree.
func (g *Graphs) Outline(root, file string) ([]*OutlineItem, error) {
	file = strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(file)), "/")
	var fileID int64
	if err := g.db.QueryRow(`SELECT id FROM code_files WHERE root = ? AND path = ?`, root, file).Scan(&fileID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errNotFound
		}
		return nil, err
	}
	syms, err := querySymbols(g.db, `SELECT `+symbolCols+` FROM code_symbols s JOIN code_files f ON f.id = s.file_id WHERE s.file_id = ? ORDER BY s.id`, fileID)
	if err != nil {
		return nil, err
	}
	items := map[int64]*OutlineItem{}
	out := []*OutlineItem{}
	for _, s := range syms {
		it := &OutlineItem{SymbolRef: s, Members: []*OutlineItem{}}
		items[s.ID] = it
		if p, ok := items[s.ParentID]; ok {
			p.Members = append(p.Members, it)
		} else {
			out = append(out, it)
		}
	}
	return out, nil
}

// Resolve maps a file and line, as an error or a profile reports them, to the
// innermost function, method or type at that line. The file may be absolute
// (path_map is applied first) or relative to a root.
func (g *Graphs) Resolve(file string, line int, rootName string) (*SymbolRef, error) {
	file = filepath.ToSlash(strings.TrimSpace(file))
	for _, m := range g.pathMap {
		from := strings.TrimRight(filepath.ToSlash(m.From), "/")
		if file == from || strings.HasPrefix(file, from+"/") {
			file = strings.TrimRight(filepath.ToSlash(m.To), "/") + strings.TrimPrefix(file, from)
			break
		}
	}
	type cand struct{ root, rel string }
	var cands []cand
	for _, r := range g.ix.Roots() {
		if rootName != "" && r.Name != rootName {
			continue
		}
		rp := filepath.ToSlash(r.Path)
		switch {
		case strings.HasPrefix(file, rp+"/"):
			cands = append(cands, cand{r.Name, strings.TrimPrefix(file, rp+"/")})
		case !strings.HasPrefix(file, "/"):
			cands = append(cands, cand{r.Name, strings.TrimPrefix(path.Clean("/"+file), "/")})
		}
	}
	for _, c := range cands {
		s, err := scanSymbol(g.db.QueryRow(`SELECT `+symbolCols+` FROM code_symbols s JOIN code_files f ON f.id = s.file_id
			WHERE f.root = ? AND f.path = ? AND s.line_start <= ? AND s.line_end >= ? AND s.kind NOT IN ('namespace', 'package')
			ORDER BY (s.line_end - s.line_start), s.id DESC LIMIT 1`, c.root, c.rel, line, line))
		if err == nil {
			return &s, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	return nil, errNotFound
}

// --- whole graph ------------------------------------------------------------

type symInfo struct {
	parent int64
	kind   string
	name   string
	fqn    string
	file   string
	line   int
}

type symbolIndex struct {
	gen  int64
	syms map[int64]*symInfo
}

func (g *Graphs) symbols(root string) (*symbolIndex, error) {
	gen, running := g.ix.generation(root)
	g.mu.Lock()
	if c, ok := g.cache[root]; ok && c.gen == gen && !running {
		g.mu.Unlock()
		return c, nil
	}
	g.mu.Unlock()
	rows, err := g.db.Query(`SELECT s.id, COALESCE(s.parent_id, 0), s.kind, s.name, s.fqn, f.path, s.line_start
		FROM code_symbols s JOIN code_files f ON f.id = s.file_id WHERE s.root = ?`, root)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idx := &symbolIndex{gen: gen, syms: map[int64]*symInfo{}}
	for rows.Next() {
		var id int64
		s := &symInfo{}
		if err := rows.Scan(&id, &s.parent, &s.kind, &s.name, &s.fqn, &s.file, &s.line); err != nil {
			return nil, err
		}
		idx.syms[id] = s
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !running {
		g.mu.Lock()
		if g.cache == nil {
			g.cache = map[string]*symbolIndex{}
		}
		g.cache[root] = idx
		g.mu.Unlock()
	}
	return idx, nil
}

// Graph levels.
const (
	LevelSymbol  = "symbol"  // functions, methods, types
	LevelClass   = "class"   // methods folded into their class or type
	LevelPackage = "package" // everything folded into its namespace or package
)

// GraphQuery selects a subgraph. With no Target and no File it covers the
// whole root.
type GraphQuery struct {
	Root      string
	Level     string
	Target    int64  // symbol id; its members are included
	File      string // all symbols of a file
	Depth     int    // hops from the target or file; ignored for the whole root
	Direction string // out, in, both
	Kinds     []string
	Limit     int
	// Expand lists node ids to show one level finer: a namespace or package
	// (ns:<fqn>) as its types and functions, a type (s:<id>) as its methods.
	Expand map[string]bool
}

var finer = map[string]string{LevelPackage: LevelClass, LevelClass: LevelSymbol}

// fold maps a symbol to its node at the query's level, one level finer for
// each expanded node on the way. group is the expanded node it sits in. ""
// for the symbol of an expanded node itself: that node is not shown.
func (x *symbolIndex) fold(id int64, level string, expand map[string]bool) (key, group string, s *symInfo) {
	key, s = x.rollup(id, level)
	if s == nil {
		return "", "", nil
	}
	for expand[key] {
		next, ok := finer[level]
		if !ok {
			// Nothing finer to show: the expand is ignored.
			return key, group, s
		}
		k, ns := x.rollup(id, next)
		if k == key || ns.kind == KindNamespace || ns.kind == KindPackage {
			// The expanded node's own symbol (the class itself, or one of
			// the namespace's per-file rows): an expanded node is never
			// shown, its own edges go.
			return "", "", nil
		}
		group, key, s, level = key, k, ns, next
	}
	return key, group, s
}

func (x *symbolIndex) rollup(id int64, level string) (string, *symInfo) {
	s := x.syms[id]
	if s == nil {
		return "", nil
	}
	switch level {
	case LevelClass:
		if s.kind == KindMethod {
			if p := x.syms[s.parent]; p != nil && p.kind != KindPackage && p.kind != KindNamespace {
				return "s:" + strconv.FormatInt(s.parent, 10), p
			}
		}
	case LevelPackage:
		cur, cid := s, id
		for i := 0; i < 8 && cur != nil; i++ {
			if cur.kind == KindNamespace || cur.kind == KindPackage {
				return "ns:" + cur.fqn, cur
			}
			cid = cur.parent
			cur = x.syms[cid]
		}
		return "ns:", &symInfo{kind: KindNamespace, name: "(global)", fqn: ""}
	}
	return "s:" + strconv.FormatInt(id, 10), s
}

// Subgraph builds a graph over resolved edges, folded to the given level.
func (g *Graphs) Subgraph(q GraphQuery) (*Graph, error) {
	if _, ok := g.ix.root(q.Root); !ok {
		return nil, errNotFound
	}
	switch q.Level {
	case LevelSymbol, LevelClass, LevelPackage:
	case "":
		q.Level = LevelPackage
		if q.Target != 0 || q.File != "" {
			q.Level = LevelSymbol
		}
	default:
		return nil, fmt.Errorf("level must be symbol, class or package")
	}
	q.Limit = clampInt(q.Limit, 300, 1, 2000)
	q.Depth = clampInt(q.Depth, 1, 1, 5)
	idx, err := g.symbols(q.Root)
	if err != nil {
		return nil, err
	}
	kinds := map[string]bool{}
	for _, k := range q.Kinds {
		if k = strings.TrimSpace(k); k != "" {
			kinds[k] = true
		}
	}

	type rawEdge struct {
		from, to int64
		kind     string
		line     int
	}
	var raw []rawEdge
	depthOf := map[int64]int{}
	load := func(query string, args ...any) error {
		rows, err := g.db.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e rawEdge
			if err := rows.Scan(&e.from, &e.to, &e.kind, &e.line); err != nil {
				return err
			}
			if len(kinds) > 0 && !kinds[e.kind] {
				continue
			}
			raw = append(raw, e)
		}
		return rows.Err()
	}

	if q.Target == 0 && q.File == "" {
		if err := load(`SELECT from_id, to_id, kind, line FROM code_edges WHERE root = ? AND from_id IS NOT NULL AND to_id IS NOT NULL`, q.Root); err != nil {
			return nil, err
		}
	} else {
		var seed []int64
		if q.Target != 0 {
			if idx.syms[q.Target] == nil {
				return nil, errNotFound
			}
			seed = append(seed, q.Target)
		} else {
			file := strings.TrimPrefix(path.Clean("/"+filepath.ToSlash(q.File)), "/")
			for id, s := range idx.syms {
				if s.file == file && s.kind != KindNamespace && s.kind != KindPackage {
					seed = append(seed, id)
				}
			}
			if len(seed) == 0 {
				return nil, errNotFound
			}
		}
		children := map[int64][]int64{}
		for id, s := range idx.syms {
			children[s.parent] = append(children[s.parent], id)
		}
		frontier := []int64{}
		for _, id := range seed {
			depthOf[id] = 0
			frontier = append(frontier, id)
			for _, c := range children[id] {
				if _, ok := depthOf[c]; !ok {
					depthOf[c] = 0
					frontier = append(frontier, c)
				}
			}
		}
		dir := q.Direction
		if dir == "" {
			dir = "both"
		}
		seenEdge := map[[2]int64]map[string]bool{}
		for level := 1; level <= q.Depth && len(frontier) > 0; level++ {
			before := len(raw)
			for i := 0; i < len(frontier); i += 400 {
				chunk := frontier[i:min(i+400, len(frontier))]
				ph := "?" + strings.Repeat(",?", len(chunk)-1)
				args := make([]any, 0, len(chunk)+1)
				args = append(args, q.Root)
				for _, id := range chunk {
					args = append(args, id)
				}
				if dir == "out" || dir == "both" {
					if err := load(`SELECT from_id, to_id, kind, line FROM code_edges WHERE root = ? AND to_id IS NOT NULL AND from_id IN (`+ph+`)`, args...); err != nil {
						return nil, err
					}
				}
				if dir == "in" || dir == "both" {
					if err := load(`SELECT from_id, to_id, kind, line FROM code_edges WHERE root = ? AND from_id IS NOT NULL AND to_id IN (`+ph+`)`, args...); err != nil {
						return nil, err
					}
				}
			}
			var next []int64
			kept := raw[:before]
			for _, e := range raw[before:] {
				k := [2]int64{e.from, e.to}
				if seenEdge[k] == nil {
					seenEdge[k] = map[string]bool{}
				}
				if seenEdge[k][e.kind+strconv.Itoa(e.line)] {
					continue
				}
				seenEdge[k][e.kind+strconv.Itoa(e.line)] = true
				kept = append(kept, e)
				for _, id := range []int64{e.from, e.to} {
					if _, ok := depthOf[id]; !ok {
						depthOf[id] = level
						next = append(next, id)
					}
				}
			}
			raw = kept
			frontier = next
		}
	}

	// Fold to the level and count.
	nodes := map[string]*GraphNode{}
	degree := map[string]int{}
	edgeIdx := map[string]*GraphEdge{}
	var edgeOrder []string
	addNode := func(id int64) string {
		key, group, s := idx.fold(id, q.Level, q.Expand)
		if s == nil {
			return ""
		}
		n, ok := nodes[key]
		d, hasDepth := depthOf[id]
		if !hasDepth {
			d = 0
		}
		if !ok {
			n = &GraphNode{ID: key, Kind: s.kind, Name: s.name, FQN: s.fqn, Depth: d, Group: group}
			if strings.HasPrefix(key, "s:") {
				n.SymbolID, _ = strconv.ParseInt(key[2:], 10, 64)
				n.File, n.Line = s.file, s.line
			}
			nodes[key] = n
		} else if hasDepth && d < n.Depth {
			n.Depth = d
		}
		return key
	}
	whole := q.Target == 0 && q.File == ""
	for _, id := range sortedKeys(depthOf) {
		addNode(id)
	}
	// Sizes; the whole-root graph also shows nodes with no edges.
	for _, id := range sortedSymbolIDs(idx) {
		s := idx.syms[id]
		if s.kind == KindNamespace || s.kind == KindPackage {
			continue
		}
		key, _, _ := idx.fold(id, q.Level, q.Expand)
		if key == "" {
			continue
		}
		if _, ok := nodes[key]; !ok && whole {
			addNode(id)
		}
		if n, ok := nodes[key]; ok {
			n.Size++
		}
	}
	for _, e := range raw {
		from, to := addNode(e.from), addNode(e.to)
		if from == "" || to == "" || from == to {
			continue
		}
		k := from + "|" + to + "|" + e.kind
		if ge, ok := edgeIdx[k]; ok {
			ge.Count++
			continue
		}
		edgeIdx[k] = &GraphEdge{From: from, To: to, Kind: e.kind, Confidence: ConfExact, Line: e.line, Count: 1}
		edgeOrder = append(edgeOrder, k)
		degree[from]++
		degree[to]++
	}

	gr := &Graph{Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	// Keep the start nodes, then the most connected ones.
	sort.Slice(keys, func(i, j int) bool {
		a, b := nodes[keys[i]], nodes[keys[j]]
		if a.Depth != b.Depth {
			return a.Depth < b.Depth
		}
		if degree[keys[i]] != degree[keys[j]] {
			return degree[keys[i]] > degree[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > q.Limit {
		keys = keys[:q.Limit]
		gr.Truncated = true
	}
	kept := map[string]bool{}
	for _, k := range keys {
		kept[k] = true
		gr.Nodes = append(gr.Nodes, *nodes[k])
	}
	for _, k := range edgeOrder {
		e := edgeIdx[k]
		if kept[e.From] && kept[e.To] {
			gr.Edges = append(gr.Edges, *e)
		}
	}
	return gr, nil
}

func sortedSymbolIDs(x *symbolIndex) []int64 {
	out := make([]int64, 0, len(x.syms))
	for k := range x.syms {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedKeys(m map[int64]int) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
