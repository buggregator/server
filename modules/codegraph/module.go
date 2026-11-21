// Package codegraph indexes source folders named in the config (PHP and Go,
// parsed with tree-sitter) into a graph of files, namespaces, types,
// functions and methods with their imports, inheritance, calls and
// instantiations. The graph lives in SQLite and is served over REST
// (/api/code/...) and MCP (code_* tools). With no roots configured the
// module is not registered: no routes but the status, no tools.
package codegraph

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed migrations/*.sql
var migrations embed.FS

// PathMap rewrites a path prefix as the app reports it (in a container) to the
// path Buggregator sees: /var/www/html -> /code.
type PathMap struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Config of the code graph.
type Config struct {
	Roots       []Root
	Exclude     []string // names or root-relative paths; default .git, node_modules, vendor
	MaxFileSize int64    // bytes; larger files are skipped. Default 1 MB
	MaxFiles    int      // per root. Default 50000
	Snippets    bool     // serve source fragments in symbol answers
	PathMap     []PathMap
	// ParseTimeout bounds one file's parse; 0 = 2s.
	ParseTimeout time.Duration
	// BaseDir is the folder the UI may browse and add roots from; "" = the
	// folder picker is off. Normalize resolves it.
	BaseDir string
	// AuthOn and Admins: with auth on only these users (username or email)
	// may browse and change the roots.
	AuthOn bool
	Admins []string
}

// DefaultExclude is used when the config names no exclude list.
var DefaultExclude = []string{".git", "node_modules", "vendor"}

// Normalize resolves root paths (absolute, symlinks followed), names roots
// without a name after their folder and drops the ones that do not exist or
// repeat a name. It never fails: a bad root is logged and left out.
func (c Config) Normalize() Config {
	out := c
	out.Roots = nil
	names := map[string]bool{}
	for _, r := range c.Roots {
		p := strings.TrimSpace(r.Path)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err == nil {
			abs, err = filepath.EvalSymlinks(abs)
		}
		if err != nil {
			slog.Warn("code graph: root skipped", "path", p, "err", err)
			continue
		}
		name := strings.TrimSpace(r.Name)
		if name == "" {
			name = filepath.Base(abs)
		}
		if names[name] {
			slog.Warn("code graph: root skipped, name repeats", "name", name, "path", p)
			continue
		}
		names[name] = true
		out.Roots = append(out.Roots, Root{Name: name, Path: abs})
	}
	out.BaseDir = resolveBase(c.BaseDir)
	if c.Exclude == nil {
		out.Exclude = DefaultExclude
	}
	if out.MaxFileSize <= 0 {
		out.MaxFileSize = 1 << 20
	}
	if out.MaxFiles <= 0 {
		out.MaxFiles = 50000
	}
	return out
}

// Module serves the code graph.
type Module struct {
	module.BaseModule
	cfg    Config
	ix     *Indexer
	graphs *Graphs
}

// New returns the module; cfg is normalized here.
func New(cfg Config) *Module {
	return &Module{cfg: cfg.Normalize()}
}

func (m *Module) Name() string { return "Code graph" }
func (m *Module) Type() string { return "code-graph" }

// Roots returns the config roots that passed normalization.
func (m *Module) Roots() []Root { return m.cfg.Roots }

// Usable reports whether the module has anything to serve: a config root or
// a base dir the UI can add roots from.
func (m *Module) Usable() bool { return len(m.cfg.Roots) > 0 || m.cfg.BaseDir != "" }

// RegisterMigrations adds the code_* tables.
func (m *Module) RegisterMigrations(migrator *storage.Migrator) error {
	return migrator.AddFromFS("code-graph", migrations, "migrations")
}

// OnInit adds the roots stored from the UI, drops data of roots no longer
// there and starts indexing every root in the background.
func (m *Module) OnInit(db *sql.DB) error {
	m.ix = newIndexer(db, m.cfg)
	m.graphs = &Graphs{db: db, ix: m.ix, snippets: m.cfg.Snippets, pathMap: m.cfg.PathMap}
	if err := m.loadUIRoots(); err != nil {
		return fmt.Errorf("code graph: %w", err)
	}
	if err := m.ix.prepare(); err != nil {
		return fmt.Errorf("code graph: %w", err)
	}
	m.ix.ReindexAll()
	return nil
}

// Indexer is exposed for tests and wiring.
func (m *Module) Indexer() *Indexer { return m.ix }

// RegisterProtectedRoutes adds /api/code routes behind the API's auth.
func (m *Module) RegisterProtectedRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	m.registerAPI(mux, auth)
}

// RegisterMCPTools adds the code_* tools.
func (m *Module) RegisterMCPTools(server *sdkmcp.Server) {
	registerTools(server, m.graphs, m.ix)
}

var _ module.ProtectedRoutesRegistrar = (*Module)(nil)
