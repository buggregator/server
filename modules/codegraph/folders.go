package codegraph

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buggregator/server/v2/internal/auth"
	"github.com/buggregator/server/v2/internal/selfupdate"
)

// Folders from the UI: the UI browses code_graph.base_dir one level at a
// time and adds folders inside it as roots. A path from the UI is relative to
// the base dir; it must resolve (symlinks followed) to a folder inside it.

var (
	errOutsideBase = errors.New("folder is outside code_graph.base_dir")
	errNotDir      = errors.New("not a folder")
)

// PickerHint says how to turn the folder picker on when it is off.
const PickerHint = "Set code_graph.base_dir (env CODE_GRAPH_BASE_DIR) to the folder Buggregator may add code from. It defaults to your home only when Buggregator listens on 127.0.0.1 or auth is on. In Docker, mount it first: -v ~/projects:/code -e CODE_GRAPH_BASE_DIR=/code"

// LocalOnlyHint says why the picker refuses a request without auth.
const LocalOnlyHint = "Without auth, code graph folders can be browsed and changed only from this computer: open the UI as http://127.0.0.1 or http://localhost. From another machine, turn on auth and list yourself in code_graph.admins."

// Picker is the folder picker's state in GET /api/code/roots.
type Picker struct {
	Enabled bool   `json:"enabled"`
	BaseDir string `json:"base_dir,omitempty"`
	// CanManage is false when auth is on and the user is not in
	// code_graph.admins: the UI shows the roots but no add, remove or browse.
	CanManage bool   `json:"can_manage"`
	Hint      string `json:"hint,omitempty"`
}

// Folder is one entry of GET /api/code/folders.
type Folder struct {
	Name string `json:"name"`
	Path string `json:"path"` // relative to base_dir, slash-separated
	// Hidden: a dot folder. Excluded: matches code_graph.exclude (vendor,
	// node_modules, .git): its files are never indexed inside a root.
	Hidden   bool   `json:"hidden"`
	Excluded bool   `json:"excluded"`
	Root     string `json:"root,omitempty"` // name of the root this folder already is
}

// FoldersResponse is the answer of GET /api/code/folders.
type FoldersResponse struct {
	BaseDir string   `json:"base_dir"`
	Path    string   `json:"path"`           // "" = base_dir itself
	Parent  *string  `json:"parent"`         // null at base_dir
	Root    string   `json:"root,omitempty"` // this folder is already a root
	Folders []Folder `json:"folders"`
}

// AddRootRequest is the body of POST /api/code/roots.
type AddRootRequest struct {
	Path string `json:"path"` // relative to base_dir
	Name string `json:"name"` // default: the folder's name
}

// resolveBase normalizes the configured base dir: absolute, symlinks
// followed, an existing folder. "" when unset or unusable (logged).
func resolveBase(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err == nil {
		abs, err = filepath.EvalSymlinks(abs)
	}
	if err == nil {
		var fi os.FileInfo
		if fi, err = os.Stat(abs); err == nil && !fi.IsDir() {
			err = errNotDir
		}
	}
	if err != nil {
		slog.Warn("code graph: base_dir unusable; folder picker off", "base_dir", p, "err", err)
		return ""
	}
	return abs
}

// inBase resolves rel (relative to base, slash-separated) to a real folder
// inside base. ".." that climbs out, an absolute path and a symlink that
// points outside are refused.
func inBase(base, rel string) (string, error) {
	if base == "" {
		return "", errOutsideBase
	}
	rel = strings.TrimSpace(rel)
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, `\`) {
		return "", errOutsideBase
	}
	joined := filepath.Join(base, filepath.FromSlash(rel))
	if !within(base, joined) {
		return "", errOutsideBase
	}
	real, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", err
	}
	if !within(base, real) {
		return "", errOutsideBase
	}
	fi, err := os.Stat(real)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", errNotDir
	}
	return real, nil
}

// relToBase is the slash path of a real folder inside base ("" for base).
func relToBase(base, real string) string {
	r, err := filepath.Rel(base, real)
	if err != nil || r == "." {
		return ""
	}
	return filepath.ToSlash(r)
}

// browse lists the folders one level below rel. Symlinks are listed under
// the link's name only when they resolve to a folder inside base.
func (m *Module) browse(rel string) (*FoldersResponse, error) {
	base := m.cfg.BaseDir
	dir, err := inBase(base, rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	byPath := map[string]string{}
	for _, r := range m.ix.Roots() {
		byPath[r.Path] = r.Name
	}
	exclude := map[string]bool{}
	for _, e := range m.cfg.Exclude {
		if !strings.Contains(e, "/") {
			exclude[e] = true
		}
	}
	here := relToBase(base, dir)
	out := &FoldersResponse{BaseDir: base, Path: here, Root: byPath[dir], Folders: []Folder{}}
	if dir != base {
		p := relToBase(base, filepath.Dir(dir))
		out.Parent = &p
	}
	for _, e := range entries {
		name := e.Name()
		full := filepath.Join(dir, name)
		if e.Type()&os.ModeSymlink != 0 {
			real, err := filepath.EvalSymlinks(full)
			if err != nil || !within(base, real) {
				continue
			}
			fi, err := os.Stat(real)
			if err != nil || !fi.IsDir() {
				continue
			}
			full = real
		} else if !e.IsDir() {
			continue
		}
		f := Folder{Name: name, Hidden: strings.HasPrefix(name, "."), Excluded: exclude[name], Root: byPath[full]}
		if full == filepath.Join(dir, name) {
			f.Path = strings.TrimPrefix(here+"/"+name, "/")
		} else {
			f.Path = relToBase(base, full)
		}
		out.Folders = append(out.Folders, f)
	}
	sort.Slice(out.Folders, func(i, j int) bool {
		return strings.ToLower(out.Folders[i].Name) < strings.ToLower(out.Folders[j].Name)
	})
	return out, nil
}

// canManage reports whether the request's user may browse and change the
// roots: with auth off only a request from this machine (loopback peer and
// Host, whatever the listen address), with auth on an admin from
// code_graph.admins.
func (m *Module) canManage(r *http.Request) bool {
	if !m.cfg.AuthOn {
		return localRequest(r)
	}
	u := auth.UserFromContext(r.Context())
	if u == nil {
		return false
	}
	for _, a := range m.cfg.Admins {
		if a != "" && (strings.EqualFold(a, u.Username) || strings.EqualFold(a, u.Email)) {
			return true
		}
	}
	return false
}

// localRequest: the peer is a loopback address and the Host names this
// machine (a DNS-rebound page has a foreign Host).
func localRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback() && selfupdate.LoopbackHost(r.Host)
}

// manage guards the picker routes: picker on, this server's own page (no
// other origin), and without auth a request from this machine only, with
// auth an admin.
func (m *Module) manage(w http.ResponseWriter, r *http.Request) bool {
	if m.cfg.BaseDir == "" {
		writeErr(w, http.StatusNotFound, "picker_off", PickerHint)
		return false
	}
	if !selfupdate.SameOrigin(r) {
		writeErr(w, http.StatusForbidden, "cross_origin", "only the Buggregator UI itself can browse or change code graph folders")
		return false
	}
	if !m.cfg.AuthOn && !localRequest(r) {
		writeErr(w, http.StatusForbidden, "not_local", LocalOnlyHint)
		return false
	}
	if !m.canManage(r) {
		writeErr(w, http.StatusForbidden, "not_admin", "only users in code_graph.admins can browse or change code graph folders")
		return false
	}
	return true
}

func (m *Module) picker(r *http.Request) Picker {
	if m.cfg.BaseDir == "" {
		return Picker{Hint: PickerHint}
	}
	p := Picker{Enabled: true, BaseDir: m.cfg.BaseDir, CanManage: m.canManage(r)}
	if !p.CanManage {
		p.BaseDir = "" // not shown to who may not browse it
		if !m.cfg.AuthOn {
			p.Hint = LocalOnlyHint
		} else {
			p.Hint = "Only users in code_graph.admins can add or remove code graph folders."
		}
	}
	return p
}

func folderErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errOutsideBase):
		writeErr(w, http.StatusBadRequest, "outside_base", err.Error())
	case errors.Is(err, errNotDir):
		writeErr(w, http.StatusBadRequest, "not_dir", err.Error())
	case errors.Is(err, os.ErrNotExist):
		writeErr(w, http.StatusNotFound, "no_such_folder", "no such folder")
	case errors.Is(err, os.ErrPermission):
		writeErr(w, http.StatusForbidden, "unreadable", "folder cannot be read")
	case errors.Is(err, errRootExists):
		writeErr(w, http.StatusConflict, "root_exists", err.Error())
	case errors.Is(err, errRootBusy):
		writeErr(w, http.StatusConflict, "root_busy", err.Error())
	case errors.Is(err, errConfigRoot):
		writeErr(w, http.StatusForbidden, "config_root", err.Error())
	case errors.Is(err, errNotFound):
		writeErr(w, http.StatusNotFound, "unknown_root", "no such root")
	default:
		writeErr(w, http.StatusInternalServerError, "internal", err.Error())
	}
}

func (m *Module) handleFolders(w http.ResponseWriter, r *http.Request) {
	if !m.manage(w, r) {
		return
	}
	res, err := m.browse(r.URL.Query().Get("path"))
	if err != nil {
		folderErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (m *Module) handleAddRoot(w http.ResponseWriter, r *http.Request) {
	if !m.manage(w, r) {
		return
	}
	var req AddRootRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_json", "body must be JSON: {\"path\": \"...\", \"name\": \"...\"}")
		return
	}
	real, err := inBase(m.cfg.BaseDir, req.Path)
	if err != nil {
		folderErr(w, err)
		return
	}
	if real == m.cfg.BaseDir {
		writeErr(w, http.StatusBadRequest, "base_root", "pick a folder inside code_graph.base_dir, not the base dir itself")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = filepath.Base(real)
	}
	if len(name) > 100 || strings.ContainsAny(name, "/\\") || strings.TrimSpace(name) == "" {
		writeErr(w, http.StatusBadRequest, "bad_name", "name: 1 to 100 characters, no slashes")
		return
	}
	st, err := m.ix.AddRoot(Root{Name: name, Path: real})
	if err != nil {
		folderErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

func (m *Module) handleRemoveRoot(w http.ResponseWriter, r *http.Request) {
	if !m.manage(w, r) {
		return
	}
	name := r.PathValue("name")
	if err := m.ix.RemoveRoot(name); err != nil {
		folderErr(w, err)
		return
	}
	m.graphs.forget(name)
	w.WriteHeader(http.StatusNoContent)
}

// loadUIRoots adds the roots stored from the UI that are still folders inside
// base_dir and do not clash with a config root. Others are skipped (logged)
// and kept in the table: they come back when base_dir allows them again.
func (m *Module) loadUIRoots() error {
	stored, err := uiRoots(m.ix.db)
	if err != nil {
		return err
	}
	for _, r := range stored {
		real, err := inBase(m.cfg.BaseDir, relToBase(m.cfg.BaseDir, r.Path))
		if err != nil || !within(m.cfg.BaseDir, r.Path) || real != r.Path {
			slog.Warn("code graph: UI root skipped, not a folder inside base_dir", "name", r.Name, "path", r.Path)
			continue
		}
		clash := false
		for _, have := range m.ix.roots {
			if have.Name == r.Name || have.Path == r.Path {
				clash = true
			}
		}
		if clash {
			slog.Warn("code graph: UI root skipped, same name or folder as a config root", "name", r.Name, "path", r.Path)
			continue
		}
		m.ix.addLocked(r)
	}
	return nil
}
