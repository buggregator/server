package mcp_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/module"
	"github.com/buggregator/server/v2/internal/storage"
)

// TestMCP_QueueJobs replays the sentry-laravel 4.31.0 queue recording
// (modules/sentry/testdata/sdk/laravel-queue) and reads it through queue_jobs.
func TestMCP_QueueJobs(t *testing.T) {
	db, store, reg := setupDB(t)
	replaySDK(t, reg, store, "laravel-queue")
	session := connectMCP(t, db, store)

	var list struct {
		Data []struct {
			ID       string `json:"id"`
			Attempts int    `json:"attempts"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(callTool(t, session, "queue_jobs", map[string]any{"status": "retried"})), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) != 1 || list.Data[0].Attempts != 2 {
		t.Fatalf("retried jobs = %+v, want one with 2 attempts", list.Data)
	}

	text := callTool(t, session, "queue_jobs", map[string]any{"id": list.Data[0].ID})
	for _, want := range []string{`"attempt_list"`, "RuntimeException", `"transaction_name": "/dispatch"`} {
		if !strings.Contains(text, want) {
			t.Errorf("job output lacks %s:\n%s", want, text)
		}
	}
}

// replaySDK feeds a recorded SDK session (modules/sentry/testdata/sdk/<dir>)
// through the ingestion handlers into the store.
func replaySDK(t *testing.T, reg *module.Registry, store *storage.SQLiteStore, dir string) {
	t.Helper()
	files, _ := filepath.Glob("../../modules/sentry/testdata/sdk/" + dir + "/*.json")
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no fixtures")
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var fx struct {
			Method, Path, Body string
			BodyB64            string            `json:"body_b64"`
			Headers            map[string]string `json:"headers"`
		}
		if err := json.Unmarshal(raw, &fx); err != nil {
			t.Fatal(err)
		}
		body := []byte(fx.Body)
		if fx.BodyB64 != "" {
			body, _ = base64.StdEncoding.DecodeString(fx.BodyB64)
		}
		req := httptest.NewRequest(fx.Method, fx.Path, bytes.NewReader(body))
		for k, v := range fx.Headers {
			req.Header.Set(k, v)
		}
		for _, h := range reg.Handlers() {
			if h.Match(req) {
				inc, err := h.Handle(req)
				if err != nil {
					t.Fatalf("%s: %v", f, err)
				}
				if inc != nil {
					if err := store.Store(context.Background(), event.NewEvent(inc)); err != nil {
						t.Fatal(err)
					}
				}
				break
			}
		}
	}
}
