package sentry

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpserver "github.com/buggregator/server/v2/internal/server/http"
	"github.com/buggregator/server/v2/internal/storage"
)

// setupProjectsDB is setupTestDB plus the core projects table, with the given
// projects already registered.
func setupProjectsDB(t *testing.T, projects ...string) *sql.DB {
	t.Helper()
	db := setupTestDB(t)
	m := storage.NewMigrator(db)
	if err := m.AddFromFS("core", storage.CoreMigrations, "migrations"); err != nil {
		t.Fatal(err)
	}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}
	for _, p := range projects {
		if _, err := db.Exec(`INSERT INTO projects (key, name) VALUES (?, ?)`, p, p); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func countProjects(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestHandle_ProjectByPublicKey: a DSN with a numeric path (required by the
// JavaScript SDKs) reaches a project through its public key, but only an
// existing one. Issue #430.
func TestHandle_ProjectByPublicKey(t *testing.T) {
	const body = `{"event_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","message":"boom","level":"error"}`
	auth := func(key string) string {
		return "Sentry sentry_version=7, sentry_client=sentry.php/4.10.0, sentry_key=" + key
	}
	for _, tc := range []struct {
		name     string
		path     string
		header   http.Header
		detected string // what the detect step found (X-Buggregator-Project, Basic Auth)
		want     string
	}{
		{name: "query key (JS) to existing project", path: "/api/1/store/?sentry_version=7&sentry_key=billing", want: "billing"},
		{name: "X-Sentry-Auth key (PHP, Python, Go) to existing project", path: "/api/1/store/", header: http.Header{"X-Sentry-Auth": {auth("billing")}}, want: "billing"},
		{name: "Authorization: Sentry key", path: "/api/1/store/", header: http.Header{"Authorization": {auth("billing")}}, want: "billing"},
		{name: "reserved key sentry goes to default", path: "/api/1/store/?sentry_key=sentry", want: ""},
		{name: "unknown key goes to default", path: "/api/1/store/", header: http.Header{"X-Sentry-Auth": {auth("0f3c9a8e5b7d4e21a6c8f0b2d4e6a8c1")}}, want: ""},
		{name: "no key goes to default", path: "/api/1/store/", want: ""},
		{name: "named path wins over the key", path: "/api/shop/store/?sentry_key=billing", want: "shop"},
		{name: "detected project wins over the key", path: "/api/1/store/?sentry_key=billing", detected: "shop", want: "shop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupProjectsDB(t, "billing")
			before := countProjects(t, db)
			h := &handler{db: db}

			r := httptest.NewRequest("POST", tc.path, strings.NewReader(body))
			for k, v := range tc.header {
				r.Header[k] = v
			}
			if tc.detected != "" {
				r.Header.Set(httpserver.HeaderDetectedProject, tc.detected)
			}
			inc, err := h.Handle(r)
			if err != nil {
				t.Fatal(err)
			}
			if inc == nil {
				t.Fatal("no event")
			}
			if inc.Project != tc.want {
				t.Errorf("project = %q, want %q", inc.Project, tc.want)
			}
			var stored string
			if err := db.QueryRow(`SELECT project_id FROM sentry_error_events`).Scan(&stored); err != nil {
				t.Fatal(err)
			}
			if stored != projectKey(tc.want) {
				t.Errorf("structured row project = %q, want %q", stored, projectKey(tc.want))
			}
			if after := countProjects(t, db); after != before {
				t.Errorf("projects = %d, want %d: a key must not create a project", after, before)
			}
		})
	}
}

// TestSDKFixtures_ProjectByPublicKey: the recorded SDK requests (DSN
// http://publickey@host/42) land in the project "publickey" once it exists.
func TestSDKFixtures_ProjectByPublicKey(t *testing.T) {
	for _, tc := range []struct{ sdk, file string }{
		{"node", "02-event.json"},   // key in the query
		{"python", "01-event.json"}, // key in X-Sentry-Auth, Brotli body
	} {
		t.Run(tc.sdk, func(t *testing.T) {
			h := &handler{db: setupProjectsDB(t, "publickey")}
			inc, err := h.Handle(sdkFixtures(t, tc.sdk)[tc.file].request(t))
			if err != nil {
				t.Fatal(err)
			}
			if inc == nil || inc.Project != "publickey" {
				t.Fatalf("event %+v, want project publickey", inc)
			}
		})
	}
}
