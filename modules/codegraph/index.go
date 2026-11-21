package codegraph

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Root is a folder whose code is indexed.
type Root struct {
	Name string `json:"name"`
	Path string `json:"path"` // absolute, symlinks resolved
	// Source is where the root comes from: SourceConfig (read-only in the
	// UI) or SourceUI (added from the UI, stored in SQLite).
	Source string `json:"source"`
}

// Root sources.
const (
	SourceConfig = "config"
	SourceUI     = "ui"
)

// Index states of a root.
const (
	StatePending  = "pending"
	StateIndexing = "indexing"
	StateReady    = "ready"
	StateError    = "error"
)

// Status is the index state of one root.
type Status struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Source      string `json:"source"` // config or ui; only ui roots can be removed
	State       string `json:"state"`
	Files       int    `json:"files"`
	Symbols     int    `json:"symbols"`
	Edges       int    `json:"edges"`
	ParseErrors int    `json:"parse_errors"`
	// Progress of the current (or last) run.
	FilesSeen   int    `json:"files_seen"`
	FilesParsed int    `json:"files_parsed"`
	Skipped     int    `json:"files_skipped"` // over max_file_size or unreadable
	Truncated   bool   `json:"truncated"`     // max_files reached
	StartedAt   string `json:"started_at,omitempty"`
	IndexedAt   string `json:"indexed_at,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
	Error       string `json:"error,omitempty"`
}

// Indexer keeps the code graph of its roots in SQLite. Roots come from the
// config and, at run time, from the UI (AddRoot, RemoveRoot).
type Indexer struct {
	db    *sql.DB
	cfg   Config
	mu    sync.Mutex
	roots []Root // in order: config roots, then UI roots
	runs  map[string]*run
	// removing are roots removed while a run was still writing them: their
	// data is dropped when that run ends, and the name is busy until then.
	removing map[string]bool
	wg       sync.WaitGroup
}

type run struct {
	root    Root
	status  Status
	running bool
	again   bool  // a reindex was asked for while running
	gen     int64 // finished runs; caches key on it
	removed bool
	cancel  context.CancelFunc
}

func newIndexer(db *sql.DB, cfg Config) *Indexer {
	ix := &Indexer{db: db, cfg: cfg, runs: map[string]*run{}, removing: map[string]bool{}}
	for _, r := range cfg.Roots {
		ix.addLocked(r)
	}
	return ix
}

func (ix *Indexer) addLocked(r Root) {
	if r.Source == "" {
		r.Source = SourceConfig
	}
	ix.roots = append(ix.roots, r)
	ix.runs[r.Name] = &run{root: r, status: Status{Name: r.Name, Path: r.Path, Source: r.Source, State: StatePending}}
}

// Roots returns the roots: config ones first, then the ones added in the UI.
func (ix *Indexer) Roots() []Root {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return append([]Root(nil), ix.roots...)
}

func (ix *Indexer) root(name string) (Root, bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	r, ok := ix.runs[name]
	if !ok {
		return Root{}, false
	}
	return r.root, true
}

// generation returns the number of finished runs of a root and whether one
// is in progress.
func (ix *Indexer) generation(name string) (int64, bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	r, ok := ix.runs[name]
	if !ok {
		return 0, false
	}
	return r.gen, r.running
}

func (ix *Indexer) parseTimeout() time.Duration {
	if ix.cfg.ParseTimeout > 0 {
		return ix.cfg.ParseTimeout
	}
	return parseTimeout
}

// Status returns the state of one root.
func (ix *Indexer) Status(name string) (Status, bool) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	r, ok := ix.runs[name]
	if !ok {
		return Status{}, false
	}
	return r.status, true
}

// Statuses returns the state of every root, in Roots order.
func (ix *Indexer) Statuses() []Status {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	out := make([]Status, 0, len(ix.roots))
	for _, r := range ix.roots {
		out = append(out, ix.runs[r.Name].status)
	}
	return out
}

// Errors of AddRoot and RemoveRoot.
var (
	errRootExists = errors.New("a root with this name or folder already exists")
	errRootBusy   = errors.New("this root is still being removed; try again in a moment")
	errConfigRoot = errors.New("roots from code_graph.roots are read-only; change the config")
)

// AddRoot adds a root (from the UI), stores it in code_ui_roots and starts
// indexing it. The caller has checked its path.
func (ix *Indexer) AddRoot(r Root) (Status, error) {
	r.Source = SourceUI
	ix.mu.Lock()
	if ix.removing[r.Name] {
		ix.mu.Unlock()
		return Status{}, errRootBusy
	}
	for _, have := range ix.roots {
		if have.Name == r.Name || have.Path == r.Path {
			ix.mu.Unlock()
			return Status{}, errRootExists
		}
	}
	ix.addLocked(r)
	ix.mu.Unlock()
	err := ix.storeRoot(r)
	if err == nil {
		_, err = ix.db.Exec(`INSERT INTO code_ui_roots (name, path, added_at) VALUES (?, ?, ?)`, r.Name, r.Path, time.Now().UTC().Format(time.RFC3339))
	}
	if err != nil {
		ix.mu.Lock()
		ix.forgetLocked(r.Name)
		ix.mu.Unlock()
		return Status{}, err
	}
	ix.Reindex(r.Name)
	st, _ := ix.Status(r.Name)
	return st, nil
}

// RemoveRoot removes a root added in the UI and its data. A run in progress
// is cancelled; its data goes when it stops.
func (ix *Indexer) RemoveRoot(name string) error {
	ix.mu.Lock()
	r, ok := ix.runs[name]
	if !ok {
		ix.mu.Unlock()
		return errNotFound
	}
	if r.root.Source != SourceUI {
		ix.mu.Unlock()
		return errConfigRoot
	}
	ix.forgetLocked(name)
	running := r.running
	if running {
		r.removed = true
		ix.removing[name] = true
		if r.cancel != nil {
			r.cancel()
		}
	}
	ix.mu.Unlock()
	if _, err := ix.db.Exec(`DELETE FROM code_ui_roots WHERE name = ?`, name); err != nil {
		return err
	}
	if running {
		return nil
	}
	return ix.dropRoot(name)
}

func (ix *Indexer) forgetLocked(name string) {
	delete(ix.runs, name)
	for i, have := range ix.roots {
		if have.Name == name {
			ix.roots = append(ix.roots[:i:i], ix.roots[i+1:]...)
			break
		}
	}
}

// uiRoots reads the roots stored by AddRoot.
func uiRoots(db *sql.DB) ([]Root, error) {
	rows, err := db.Query(`SELECT name, path FROM code_ui_roots ORDER BY added_at, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Root
	for rows.Next() {
		var r Root
		if err := rows.Scan(&r.Name, &r.Path); err != nil {
			return nil, err
		}
		r.Source = SourceUI
		out = append(out, r)
	}
	return out, rows.Err()
}

func (ix *Indexer) storeRoot(r Root) error {
	_, err := ix.db.Exec(`INSERT INTO code_roots (name, path) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET path = excluded.path`, r.Name, r.Path)
	return err
}

// prepare drops the data of roots no longer in the config and loads the
// counts of the ones that are, so the status is right before the first run.
func (ix *Indexer) prepare() error {
	names := map[string]bool{}
	for _, r := range ix.roots {
		names[r.Name] = true
	}
	rows, err := ix.db.Query(`SELECT name FROM code_roots`)
	if err != nil {
		return err
	}
	var gone []string
	for rows.Next() {
		var n string
		if rows.Scan(&n) == nil && !names[n] {
			gone = append(gone, n)
		}
	}
	rows.Close()
	for _, n := range gone {
		if err := ix.dropRoot(n); err != nil {
			return err
		}
	}
	for _, r := range ix.roots {
		if err := ix.storeRoot(r); err != nil {
			return err
		}
		var indexedAt string
		_ = ix.db.QueryRow(`SELECT indexed_at FROM code_roots WHERE name = ?`, r.Name).Scan(&indexedAt)
		ix.mu.Lock()
		st := &ix.runs[r.Name].status
		st.IndexedAt = indexedAt
		ix.mu.Unlock()
		ix.loadCounts(r.Name)
	}
	return nil
}

func (ix *Indexer) dropRoot(name string) error {
	for _, q := range []string{
		`DELETE FROM code_edges WHERE root = ?`,
		`DELETE FROM code_symbols WHERE root = ?`,
		`DELETE FROM code_files WHERE root = ?`,
		`DELETE FROM code_roots WHERE name = ?`,
	} {
		if _, err := ix.db.Exec(q, name); err != nil {
			return err
		}
	}
	return nil
}

func (ix *Indexer) loadCounts(name string) {
	var files, symbols, edges, perr int
	_ = ix.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(parse_error), 0) FROM code_files WHERE root = ?`, name).Scan(&files, &perr)
	_ = ix.db.QueryRow(`SELECT COUNT(*) FROM code_symbols WHERE root = ?`, name).Scan(&symbols)
	_ = ix.db.QueryRow(`SELECT COUNT(*) FROM code_edges WHERE root = ?`, name).Scan(&edges)
	ix.mu.Lock()
	defer ix.mu.Unlock()
	r, ok := ix.runs[name]
	if !ok {
		return
	}
	st := &r.status
	st.Files, st.Symbols, st.Edges, st.ParseErrors = files, symbols, edges, perr
}

// Reindex starts a background run for a root. A run already in progress is
// not interrupted; one more run follows it. It returns false for an unknown
// root.
func (ix *Indexer) Reindex(name string) bool {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	r, ok := ix.runs[name]
	if !ok {
		return false
	}
	if r.running {
		r.again = true
		return true
	}
	r.running = true
	r.status.State = StateIndexing
	ix.wg.Add(1)
	go ix.loop(r)
	return true
}

// ReindexAll starts a run for every root.
func (ix *Indexer) ReindexAll() {
	for _, r := range ix.Roots() {
		ix.Reindex(r.Name)
	}
}

// Wait blocks until no run is in progress (tests, shutdown).
func (ix *Indexer) Wait() { ix.wg.Wait() }

func (ix *Indexer) loop(r *run) {
	defer ix.wg.Done()
	for {
		ctx, cancel := context.WithCancel(context.Background())
		ix.mu.Lock()
		r.cancel = cancel
		removed := r.removed
		ix.mu.Unlock()
		var err error
		if !removed {
			err = ix.index(ctx, r.root)
		}
		cancel()
		ix.mu.Lock()
		removed = r.removed
		ix.mu.Unlock()
		if removed {
			if err := ix.dropRoot(r.root.Name); err != nil {
				slog.Warn("code graph: dropping a removed root failed", "root", r.root.Name, "err", err)
			}
			ix.mu.Lock()
			delete(ix.removing, r.root.Name)
			r.running = false
			ix.mu.Unlock()
			return
		}
		if err != nil {
			slog.Warn("code graph: indexing failed", "root", r.root.Name, "err", err)
		}
		ix.loadCounts(r.root.Name)
		ix.mu.Lock()
		r.gen++
		if err != nil {
			r.status.State, r.status.Error = StateError, err.Error()
		} else {
			r.status.State, r.status.Error = StateReady, ""
		}
		if !r.again {
			r.running = false
			ix.mu.Unlock()
			return
		}
		r.again = false
		r.status.State = StateIndexing
		ix.mu.Unlock()
	}
}

func (ix *Indexer) progress(name string, fn func(s *Status)) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if r, ok := ix.runs[name]; ok {
		fn(&r.status)
	}
}

type diskFile struct {
	rel   string // slash-separated, relative to the root
	lang  string
	size  int64
	mtime int64
}

type dbFile struct {
	id    int64
	size  int64
	mtime int64
	hash  string
}

// excluded reports whether a directory or file name matches the exclude list.
// A pattern with a slash matches the path relative to the root, one without
// matches any path element.
func (ix *Indexer) excluded(rel, name string) bool {
	for _, p := range ix.cfg.Exclude {
		if strings.Contains(p, "/") {
			p = strings.Trim(p, "/")
			if ok, _ := path.Match(p, rel); ok || rel == p || strings.HasPrefix(rel, p+"/") {
				return true
			}
			continue
		}
		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}
	return false
}

// walk lists the source files of a root and the Go modules in it
// (directory -> module path). Symlinks are not followed.
func (ix *Indexer) walk(ctx context.Context, root Root) ([]diskFile, map[string]string, int, bool, error) {
	var files []diskFile
	modules := map[string]string{}
	skipped, truncated := 0, false
	err := filepath.WalkDir(root.Path, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if p == root.Path {
				return err
			}
			skipped++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root.Path, p)
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if ix.excluded(rel, d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || ix.excluded(rel, d.Name()) {
			return nil
		}
		if d.Name() == "go.mod" {
			if mod := readModulePath(p); mod != "" {
				modules[path.Dir(rel)] = mod
			}
			return nil
		}
		lang := language(d.Name())
		if lang == "" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			skipped++
			return nil
		}
		if info.Size() > ix.cfg.MaxFileSize {
			skipped++
			return nil
		}
		if len(files) >= ix.cfg.MaxFiles {
			truncated = true
			return fs.SkipAll
		}
		files = append(files, diskFile{rel: rel, lang: lang, size: info.Size(), mtime: info.ModTime().UnixNano()})
		return nil
	})
	return files, modules, skipped, truncated, err
}

func readModulePath(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if rest, ok := strings.CutPrefix(line, "module"); ok && (rest == "" || rest[0] == ' ' || rest[0] == '\t') {
			return strings.Trim(strings.TrimSpace(rest), `"`)
		}
	}
	return ""
}

// goPackage is the import path of a Go file's directory: the nearest go.mod's
// module path plus the directory below it; without go.mod, the directory.
func goPackage(rootName, rel string, modules map[string]string) string {
	dir := path.Dir(rel)
	for d := dir; ; d = path.Dir(d) {
		if mod, ok := modules[d]; ok {
			if d == dir {
				return mod
			}
			sub := strings.TrimPrefix(dir, d+"/")
			if d == "." {
				sub = dir
			}
			return mod + "/" + sub
		}
		if d == "." || d == "/" {
			break
		}
	}
	if dir == "." {
		return rootName
	}
	return dir
}

type parsedRow struct {
	disk   diskFile
	hash   string
	oldID  int64
	result *parsedFile
}

const writeBatch = 100

// index brings one root's rows in line with the disk: new and changed files
// are parsed, vanished ones deleted, then the edges are resolved again.
func (ix *Indexer) index(ctx context.Context, root Root) error {
	started := time.Now()
	ix.progress(root.Name, func(s *Status) {
		s.StartedAt = started.UTC().Format(time.RFC3339)
		s.FilesSeen, s.FilesParsed, s.Skipped, s.Truncated = 0, 0, 0, false
	})

	files, modules, skipped, truncated, err := ix.walk(ctx, root)
	if err != nil {
		return fmt.Errorf("read %s: %w", root.Path, err)
	}
	ix.progress(root.Name, func(s *Status) { s.Skipped, s.Truncated = skipped, truncated })

	known := map[string]dbFile{}
	rows, err := ix.db.Query(`SELECT id, path, size, mtime, hash FROM code_files WHERE root = ?`, root.Name)
	if err != nil {
		return err
	}
	for rows.Next() {
		var f dbFile
		var p string
		if err := rows.Scan(&f.id, &p, &f.size, &f.mtime, &f.hash); err != nil {
			rows.Close()
			return err
		}
		known[p] = f
	}
	rows.Close()

	ps := newParsers(ix.parseTimeout())
	defer ps.watch(ctx)()
	var batch []parsedRow
	var touched [][3]int64 // id, size, mtime: same content, new mtime
	seen := make(map[string]bool, len(files))
	for i, f := range files {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		seen[f.rel] = true
		old, had := known[f.rel]
		if had && old.size == f.size && old.mtime == f.mtime {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root.Path, filepath.FromSlash(f.rel)))
		if err != nil {
			skipped++
			continue
		}
		sum := sha256.Sum256(src)
		hash := hex.EncodeToString(sum[:])
		if had && old.hash == hash {
			touched = append(touched, [3]int64{old.id, f.size, f.mtime})
			continue
		}
		res, err := ps.parse(f.lang, src, goPackage(root.Name, f.rel, modules))
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			slog.Warn("code graph: file not parsed", "root", root.Name, "file", f.rel, "err", err)
			res = &parsedFile{HasError: true}
		} else if res.Stopped != "" {
			slog.Warn("code graph: syntax error recovery stopped, file kept as parsed so far", "root", root.Name, "file", f.rel, "reason", res.Stopped)
		}
		batch = append(batch, parsedRow{disk: f, hash: hash, oldID: old.id, result: res})
		if len(batch) >= writeBatch {
			if err := ix.write(root.Name, batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
		n := i + 1
		ix.progress(root.Name, func(s *Status) { s.FilesSeen, s.FilesParsed = n, s.FilesParsed+1 })
	}
	if err := ix.write(root.Name, batch); err != nil {
		return err
	}
	ix.progress(root.Name, func(s *Status) { s.FilesSeen, s.Skipped = len(files), skipped })

	var gone []int64
	for p, f := range known {
		if !seen[p] {
			gone = append(gone, f.id)
		}
	}
	if err := ix.touchAndDelete(touched, gone); err != nil {
		return err
	}
	if err := ix.resolve(root.Name); err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	indexedAt := time.Now().UTC().Format(time.RFC3339)
	if _, err := ix.db.Exec(`UPDATE code_roots SET indexed_at = ?, error = '' WHERE name = ?`, indexedAt, root.Name); err != nil {
		return err
	}
	ix.progress(root.Name, func(s *Status) {
		s.IndexedAt = indexedAt
		s.DurationMs = time.Since(started).Milliseconds()
	})
	return nil
}

// write replaces the rows of a batch of parsed files in one transaction.
func (ix *Indexer) write(rootName string, batch []parsedRow) error {
	if len(batch) == 0 {
		return nil
	}
	tx, err := ix.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	insSym, err := tx.Prepare(`INSERT INTO code_symbols (root, file_id, parent_id, kind, name, fqn, line_start, line_end, signature) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insSym.Close()
	insEdge, err := tx.Prepare(`INSERT INTO code_edges (root, file_id, from_id, kind, to_name, to_fqn, alt_fqn, to_kind, line, confidence) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer insEdge.Close()

	for _, row := range batch {
		fileID := row.oldID
		perr := 0
		if row.result.HasError {
			perr = 1
		}
		if fileID != 0 {
			if _, err := tx.Exec(`DELETE FROM code_edges WHERE file_id = ?`, fileID); err != nil {
				return err
			}
			if _, err := tx.Exec(`DELETE FROM code_symbols WHERE file_id = ?`, fileID); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE code_files SET lang = ?, size = ?, mtime = ?, hash = ?, parse_error = ? WHERE id = ?`,
				row.disk.lang, row.disk.size, row.disk.mtime, row.hash, perr, fileID); err != nil {
				return err
			}
		} else {
			res, err := tx.Exec(`INSERT INTO code_files (root, path, lang, size, mtime, hash, parse_error) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				rootName, row.disk.rel, row.disk.lang, row.disk.size, row.disk.mtime, row.hash, perr)
			if err != nil {
				return err
			}
			if fileID, err = res.LastInsertId(); err != nil {
				return err
			}
		}
		ids := make([]int64, len(row.result.Symbols))
		for i, s := range row.result.Symbols {
			var parent any
			if s.Parent >= 0 && s.Parent < i {
				parent = ids[s.Parent]
			}
			res, err := insSym.Exec(rootName, fileID, parent, s.Kind, s.Name, s.FQN, s.LineStart, s.LineEnd, s.Signature)
			if err != nil {
				return err
			}
			if ids[i], err = res.LastInsertId(); err != nil {
				return err
			}
		}
		for _, e := range row.result.Edges {
			var from any
			if e.From >= 0 && e.From < len(ids) {
				from = ids[e.From]
			}
			if _, err := insEdge.Exec(rootName, fileID, from, e.Kind, e.ToName, e.ToFQN, e.AltFQN, e.ToKind, e.Line, e.Confidence); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (ix *Indexer) touchAndDelete(touched [][3]int64, gone []int64) error {
	if len(touched) == 0 && len(gone) == 0 {
		return nil
	}
	tx, err := ix.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, t := range touched {
		if _, err := tx.Exec(`UPDATE code_files SET size = ?, mtime = ? WHERE id = ?`, t[1], t[2], t[0]); err != nil {
			return err
		}
	}
	for _, id := range gone {
		for _, q := range []string{`DELETE FROM code_edges WHERE file_id = ?`, `DELETE FROM code_symbols WHERE file_id = ?`, `DELETE FROM code_files WHERE id = ?`} {
			if _, err := tx.Exec(q, id); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// resolve links edges to symbols after the rows changed: exact names, the PHP
// global function fallback, then methods inherited from a parent class or a
// trait. Go methods are hung under their type, which may sit in another file.
func (ix *Indexer) resolve(rootName string) error {
	steps := []string{
		`UPDATE code_symbols SET parent_id = COALESCE(
			(SELECT t.id FROM code_symbols t WHERE t.root = code_symbols.root
				AND t.fqn = substr(code_symbols.fqn, 1, length(code_symbols.fqn) - length(code_symbols.name) - 1)
				AND t.kind IN ('struct', 'type', 'interface') ORDER BY t.id LIMIT 1),
			(SELECT p.id FROM code_symbols p WHERE p.file_id = code_symbols.file_id AND p.kind = 'package' LIMIT 1))
		WHERE root = ? AND kind = 'method' AND instr(fqn, '::') = 0`,
		`UPDATE code_edges SET to_id = (SELECT s.id FROM code_symbols s WHERE s.root = code_edges.root AND s.fqn = code_edges.to_fqn ORDER BY s.id LIMIT 1)
		WHERE root = ? AND confidence = 'exact' AND to_fqn != ''`,
		`UPDATE code_edges SET to_id = (SELECT s.id FROM code_symbols s WHERE s.root = code_edges.root AND s.fqn = code_edges.alt_fqn ORDER BY s.id LIMIT 1)
		WHERE root = ? AND to_id IS NULL AND alt_fqn != ''`,
	}
	for _, q := range steps {
		if _, err := ix.db.Exec(q, rootName); err != nil {
			return err
		}
	}
	return ix.resolveInherited(rootName)
}

func (ix *Indexer) resolveInherited(rootName string) error {
	type pending struct {
		id         int64
		class, met string
	}
	var todo []pending
	rows, err := ix.db.Query(`SELECT id, to_fqn FROM code_edges WHERE root = ? AND to_id IS NULL AND confidence = 'exact' AND to_kind = 'method' AND instr(to_fqn, '::') > 0`, rootName)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var fqn string
		if rows.Scan(&id, &fqn) == nil {
			class, met, _ := strings.Cut(fqn, "::")
			todo = append(todo, pending{id, class, met})
		}
	}
	rows.Close()
	if len(todo) == 0 {
		return nil
	}

	parents := map[string][]string{}
	rows, err = ix.db.Query(`SELECT f.fqn, e.to_fqn FROM code_edges e JOIN code_symbols f ON f.id = e.from_id
		WHERE e.root = ? AND e.kind IN ('extends', 'uses') AND e.to_fqn != ''`, rootName)
	if err != nil {
		return err
	}
	for rows.Next() {
		var from, to string
		if rows.Scan(&from, &to) == nil {
			parents[from] = append(parents[from], to)
		}
	}
	rows.Close()

	methods := map[string]int64{}
	rows, err = ix.db.Query(`SELECT fqn, MIN(id) FROM code_symbols WHERE root = ? AND kind = 'method' AND instr(fqn, '::') > 0 GROUP BY fqn`, rootName)
	if err != nil {
		return err
	}
	for rows.Next() {
		var fqn string
		var id int64
		if rows.Scan(&fqn, &id) == nil {
			methods[fqn] = id
		}
	}
	rows.Close()

	type hit struct{ edge, sym int64 }
	var hits []hit
	for _, p := range todo {
		queue, visited := []string{p.class}, map[string]bool{p.class: true}
		for depth := 0; len(queue) > 0 && depth < 10; depth++ {
			var next []string
			found := false
			for _, c := range queue {
				for _, parent := range parents[c] {
					if id, ok := methods[parent+"::"+p.met]; ok {
						hits = append(hits, hit{p.id, id})
						found = true
						break
					}
					if !visited[parent] {
						visited[parent] = true
						next = append(next, parent)
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
			queue = next
		}
	}
	if len(hits) == 0 {
		return nil
	}
	tx, err := ix.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck
	for _, h := range hits {
		if _, err := tx.Exec(`UPDATE code_edges SET to_id = ? WHERE id = ?`, h.sym, h.edge); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// readSource returns lines [from, to] of a file inside a root. The resolved
// path must stay inside the root.
func readSource(root Root, rel string, from, to int) (string, error) {
	full, err := safeJoin(root, rel)
	if err != nil {
		return "", err
	}
	src, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	lines := bytes.Split(src, []byte("\n"))
	if from < 1 {
		from = 1
	}
	if to > len(lines) {
		to = len(lines)
	}
	if from > to {
		return "", nil
	}
	return string(bytes.Join(lines[from-1:to], []byte("\n"))), nil
}

var errOutsideRoot = errors.New("path is outside the root")

// safeJoin resolves rel inside root (symlinks too) and refuses anything that
// lands outside it.
func safeJoin(root Root, rel string) (string, error) {
	clean := filepath.Clean(filepath.Join(root.Path, filepath.FromSlash(rel)))
	real, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", err
	}
	if !within(root.Path, real) {
		return "", errOutsideRoot
	}
	return real, nil
}

func within(root, p string) bool {
	r, err := filepath.Rel(root, p)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}
