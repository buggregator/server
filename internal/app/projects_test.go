package app

import (
	"net/http"
	"testing"

	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
)

// TestSeedProjects_AfterMigrations: projects from the config file exist right
// after start on a fresh in-memory database (issue #432).
func TestSeedProjects_AfterMigrations(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	// The start-up order of App.Run: migrations first, then the seed.
	if err := module.NewRegistry().Init(db, http.NewServeMux(), nil); err != nil {
		t.Fatal(err)
	}
	seedProjects(db, []ProjectDef{{Key: "my-app", Name: "My App"}, {Key: "default", Name: "Default"}})

	var name string
	if err := db.QueryRow(`SELECT name FROM projects WHERE key = 'my-app'`).Scan(&name); err != nil {
		t.Fatalf("project my-app: %v", err)
	}
	if name != "My App" {
		t.Errorf("name = %q, want %q", name, "My App")
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n)
	if n != 2 {
		t.Errorf("projects = %d, want 2 (default + my-app)", n)
	}
}

// TestSeedProjects_NoTable: without the projects table the seed logs and goes
// on; it does not stop the start.
func TestSeedProjects_NoTable(t *testing.T) {
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	seedProjects(db, []ProjectDef{{Key: "my-app", Name: "My App"}})
}
