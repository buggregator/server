package xdebugtrace

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// registerAPI adds the /api/xdebug-traces routes; protect is the API auth.
//
//	GET /api/xdebug-traces/{uuid}/summary            summary with the call index
//	GET /api/xdebug-traces/{uuid}/tree?parent=&from=&limit=
//	                                                 one level of calls: top level with
//	                                                 no parent, callees of the entry at
//	                                                 offset parent, or a next page (from)
//	GET /api/xdebug-traces/{uuid}/search?q=&from=&limit=
func registerAPI(mux *http.ServeMux, m *Module, protect func(http.Handler) http.Handler) {
	route := func(pattern string, h http.HandlerFunc) { mux.Handle(pattern, protect(h)) }
	route("GET /api/xdebug-traces/{uuid}/summary", m.handleSummary)
	route("GET /api/xdebug-traces/{uuid}/tree", m.handleTree)
	route("GET /api/xdebug-traces/{uuid}/search", m.handleSearch)
}

func (m *Module) handleSummary(w http.ResponseWriter, r *http.Request) {
	uuid := r.PathValue("uuid")
	p, err := m.payload(r.Context(), uuid)
	if err != nil {
		writeError(w, http.StatusNotFound, "trace not found")
		return
	}
	sum := *p.Summary
	if b, err := os.ReadFile(filepath.Join(m.dir, uuid+extIndex)); err == nil {
		_ = json.Unmarshal(b, &sum.Index)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"file": p.File, "name": p.Name, "url": p.URL, "size": p.Size, "compressed": p.Compressed,
		"summary": sum,
	})
}

func (m *Module) handleTree(w http.ResponseWriter, r *http.Request) {
	parent, ok1 := offsetParam(r, "parent")
	from, ok2 := offsetParam(r, "from")
	limit, ok3 := limitParam(r)
	if !ok1 || !ok2 || !ok3 {
		writeError(w, http.StatusBadRequest, "parent, from and limit must be whole numbers")
		return
	}
	m.scan(w, r, func(f *os.File) (*Page, error) {
		return Children(f, m.limits.MaxSize, parent, from, limit)
	})
}

func (m *Module) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" || len(q) > 256 {
		writeError(w, http.StatusBadRequest, "q is required (at most 256 bytes)")
		return
	}
	from, ok1 := offsetParam(r, "from")
	limit, ok2 := limitParam(r)
	if !ok1 || !ok2 {
		writeError(w, http.StatusBadRequest, "from and limit must be whole numbers")
		return
	}
	m.scan(w, r, func(f *os.File) (*Page, error) {
		return Search(f, m.limits.MaxSize, q, max(from, 0), limit)
	})
}

func (m *Module) scan(w http.ResponseWriter, r *http.Request, run func(*os.File) (*Page, error)) {
	uuid := r.PathValue("uuid")
	p, err := m.payload(r.Context(), uuid)
	if err != nil {
		writeError(w, http.StatusNotFound, "trace not found")
		return
	}
	f, err := os.Open(filepath.Join(m.dir, p.File))
	if err != nil {
		writeError(w, http.StatusNotFound, "trace file is gone")
		return
	}
	defer f.Close()
	page, err := run(f)
	switch {
	case errors.Is(err, ErrBadOffset), errors.Is(err, ErrTooLarge):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, "cannot read trace: "+err.Error())
	default:
		writeJSON(w, http.StatusOK, page)
	}
}

// offsetParam reads a byte offset; absent = -1.
func offsetParam(r *http.Request, name string) (int64, bool) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return -1, true
	}
	n, err := strconv.ParseInt(v, 10, 64)
	return n, err == nil && n >= 0
}

func limitParam(r *http.Request) (int, bool) {
	v := r.URL.Query().Get("limit")
	if v == "" {
		return 200, true
	}
	n, err := strconv.Atoi(v)
	return n, err == nil && n > 0
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
