package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

// Migration represents a single SQL migration file.
type Migration struct {
	Name    string // e.g., "2024_01_01_000000_create_events_table.sql"
	Module  string // e.g., "core", "profiler"
	Content string
}

// Migrator collects and runs SQL migrations from all modules.
type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// AddFromFS registers all .sql files from an embedded filesystem.
// Files are expected to be named like: 2024_01_01_000000_description.sql
func (m *Migrator) AddFromFS(moduleName string, fsys embed.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s/%s: %w", moduleName, dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(fsys, dir+"/"+entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s/%s: %w", moduleName, entry.Name(), err)
		}

		m.migrations = append(m.migrations, Migration{
			Name:    entry.Name(),
			Module:  moduleName,
			Content: string(content),
		})
	}
	return nil
}

// AddSQL registers a raw SQL migration with a given name and module.
func (m *Migrator) AddSQL(moduleName, name, content string) {
	m.migrations = append(m.migrations, Migration{
		Name:    name,
		Module:  moduleName,
		Content: content,
	})
}

// Run executes all collected migrations sorted by filename.
// Uses a migrations tracking table to avoid re-running.
func (m *Migrator) Run() error {
	// Create migrations tracking table.
	if _, err := m.db.Exec(`CREATE TABLE IF NOT EXISTS migrations (
		name TEXT PRIMARY KEY,
		module TEXT NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	// Sort all migrations by name (date-based naming ensures correct order).
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Name < m.migrations[j].Name
	})

	// Run each migration.
	for _, mig := range m.migrations {
		// Check if already applied.
		var count int
		m.db.QueryRow(`SELECT COUNT(*) FROM migrations WHERE name = ?`, mig.Module+"/"+mig.Name).Scan(&count)
		if count > 0 {
			continue
		}

		slog.Info("running migration", "module", mig.Module, "name", mig.Name)

		// Execute all statements in the migration file.
		stmts := splitStatements(mig.Content)
		for _, stmt := range stmts {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := m.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %s/%s failed: %w\nSQL: %s", mig.Module, mig.Name, err, stmt)
			}
		}

		// Mark as applied.
		if _, err := m.db.Exec(
			`INSERT INTO migrations (name, module) VALUES (?, ?)`,
			mig.Module+"/"+mig.Name, mig.Module,
		); err != nil {
			return fmt.Errorf("record migration %s/%s: %w", mig.Module, mig.Name, err)
		}
	}

	return nil
}

// splitStatements splits SQL content by semicolons, respecting basic quoting.
func splitStatements(sql string) []string {
	var stmts []string
	current := strings.Builder{}
	inString := false
	var quote byte

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if inString {
			current.WriteByte(ch)
			if ch == quote {
				inString = false
			}
			continue
		}

		if ch == '\'' || ch == '"' {
			inString = true
			quote = ch
			current.WriteByte(ch)
			continue
		}

		if ch == ';' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				stmts = append(stmts, s)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	s := strings.TrimSpace(current.String())
	if s != "" {
		stmts = append(stmts, s)
	}

	return stmts
}
