package codegraph

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type apiError struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // PHP names (App\Foo::bar, ->) stay readable
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiError{Code: code, Error: msg})
}

// fail maps a query error to a response.
func fail(w http.ResponseWriter, err error) {
	if errors.Is(err, errNotFound) {
		writeErr(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, "internal", err.Error())
}

// RootsResponse is the answer of GET /api/code/roots.
type RootsResponse struct {
	Enabled  bool     `json:"enabled"`
	Snippets bool     `json:"snippets"`
	Roots    []Status `json:"roots"`
}

func (m *Module) registerAPI(mux *http.ServeMux, wrap func(http.Handler) http.Handler) {
	h := func(pattern string, fn http.HandlerFunc) {
		mux.Handle(pattern, wrap(fn))
	}
	h("GET /api/code/roots", m.handleRoots)
	h("GET /api/code/roots/{name}", m.handleRoot)
	h("POST /api/code/roots/{name}/reindex", m.handleReindex)
	h("GET /api/code/search", m.handleSearch)
	h("GET /api/code/symbols/{id}", m.withSymbol(m.handleSymbol))
	h("GET /api/code/symbols/{id}/callers", m.withSymbol(m.handleCalls(true)))
	h("GET /api/code/symbols/{id}/callees", m.withSymbol(m.handleCalls(false)))
	h("GET /api/code/files", m.handleFiles)
	h("GET /api/code/files/outline", m.handleOutline)
	h("GET /api/code/graph", m.handleGraph)
	h("GET /api/code/resolve", m.handleResolve)
}

func (m *Module) handleRoots(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, RootsResponse{Enabled: true, Snippets: m.cfg.Snippets, Roots: m.ix.Statuses()})
}

func (m *Module) handleRoot(w http.ResponseWriter, r *http.Request) {
	st, ok := m.ix.Status(r.PathValue("name"))
	if !ok {
		writeErr(w, http.StatusNotFound, "unknown_root", "no such root in code_graph.roots")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (m *Module) handleReindex(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !m.ix.Reindex(name) {
		writeErr(w, http.StatusNotFound, "unknown_root", "no such root in code_graph.roots")
		return
	}
	st, _ := m.ix.Status(name)
	writeJSON(w, http.StatusAccepted, st)
}

func intParam(r *http.Request, name string) int {
	v, _ := strconv.Atoi(r.URL.Query().Get(name))
	return v
}

// rootParam checks an optional ?root= against the config.
func (m *Module) rootParam(w http.ResponseWriter, r *http.Request, required bool) (string, bool) {
	name := strings.TrimSpace(r.URL.Query().Get("root"))
	if name == "" {
		if required {
			if len(m.cfg.Roots) == 1 {
				return m.cfg.Roots[0].Name, true
			}
			writeErr(w, http.StatusBadRequest, "root_required", "root is required when more than one root is configured")
			return "", false
		}
		return "", true
	}
	if _, ok := m.ix.root(name); !ok {
		writeErr(w, http.StatusNotFound, "unknown_root", "no such root in code_graph.roots")
		return "", false
	}
	return name, true
}

func (m *Module) handleSearch(w http.ResponseWriter, r *http.Request) {
	root, ok := m.rootParam(w, r, false)
	if !ok {
		return
	}
	q := r.URL.Query()
	res, err := m.graphs.Search(q.Get("q"), q.Get("kind"), root, intParam(r, "limit"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbols": res})
}

func (m *Module) withSymbol(fn func(w http.ResponseWriter, r *http.Request, id int64)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || id <= 0 {
			writeErr(w, http.StatusBadRequest, "bad_id", "symbol id must be a positive number")
			return
		}
		fn(w, r, id)
	}
}

func (m *Module) handleSymbol(w http.ResponseWriter, _ *http.Request, id int64) {
	d, err := m.graphs.Detail(id)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (m *Module) handleCalls(incoming bool) func(w http.ResponseWriter, r *http.Request, id int64) {
	return func(w http.ResponseWriter, r *http.Request, id int64) {
		g, err := m.graphs.Calls(id, incoming, intParam(r, "depth"), intParam(r, "limit"))
		if err != nil {
			fail(w, err)
			return
		}
		writeJSON(w, http.StatusOK, g)
	}
}

func (m *Module) handleFiles(w http.ResponseWriter, r *http.Request) {
	root, ok := m.rootParam(w, r, true)
	if !ok {
		return
	}
	entries, err := m.graphs.Files(root, r.URL.Query().Get("path"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"root": root, "entries": entries})
}

func (m *Module) handleOutline(w http.ResponseWriter, r *http.Request) {
	root, ok := m.rootParam(w, r, true)
	if !ok {
		return
	}
	file := r.URL.Query().Get("path")
	if file == "" {
		writeErr(w, http.StatusBadRequest, "path_required", "path is required")
		return
	}
	items, err := m.graphs.Outline(root, file)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"root": root, "path": file, "symbols": items})
}

func (m *Module) handleGraph(w http.ResponseWriter, r *http.Request) {
	root, ok := m.rootParam(w, r, true)
	if !ok {
		return
	}
	q := r.URL.Query()
	gq := GraphQuery{Root: root, Level: q.Get("level"), File: q.Get("file"), Depth: intParam(r, "depth"), Direction: q.Get("direction"), Limit: intParam(r, "limit")}
	if k := q.Get("kinds"); k != "" {
		gq.Kinds = strings.Split(k, ",")
	}
	switch gq.Direction {
	case "", "in", "out", "both":
	default:
		writeErr(w, http.StatusBadRequest, "bad_direction", "direction must be in, out or both")
		return
	}
	if t := strings.TrimSpace(q.Get("target")); t != "" {
		s, err := m.graphs.Lookup(t, root)
		if err != nil {
			fail(w, err)
			return
		}
		gq.Target = s.ID
	}
	g, err := m.graphs.Subgraph(gq)
	if err != nil {
		if strings.HasPrefix(err.Error(), "level must") {
			writeErr(w, http.StatusBadRequest, "bad_level", err.Error())
			return
		}
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (m *Module) handleResolve(w http.ResponseWriter, r *http.Request) {
	root, ok := m.rootParam(w, r, false)
	if !ok {
		return
	}
	file := r.URL.Query().Get("file")
	line := intParam(r, "line")
	if file == "" || line <= 0 {
		writeErr(w, http.StatusBadRequest, "file_and_line_required", "file and a positive line are required")
		return
	}
	s, err := m.graphs.Resolve(file, line, root)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}
