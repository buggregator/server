package mcp_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/storage"
)

// An agent lists machines with events_facets, then filters events_list by one.
func TestMCP_EventsFacetsAndFilter(t *testing.T) {
	db, store, reg := setupDB(t)
	if err := storage.InitFacets(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	store.SetFacetFunc(reg.Previews().Facets)
	session := connectMCP(t, db, store)

	ingest := func(host string) string {
		body := `{"profile":{"main()":{"wt":1,"cpu":1,"mu":1,"pmu":1,"ct":1}},"app_name":"app","hostname":"` + host + `","date":1711800000,"tags":{"route":"/cart"}}`
		req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(body)))
		req.Header.Set("X-Profiler-Dump", "1")
		req.Header.Set("Content-Type", "application/json")
		return ingestViaHandler(t, reg, store, req)
	}
	web1 := ingest("web-1")
	ingest("web-2")
	ingest("web-2")

	var counts map[string][]event.FacetCount
	if err := json.Unmarshal([]byte(callTool(t, session, "events_facets", map[string]any{})), &counts); err != nil {
		t.Fatal(err)
	}
	if got := counts["host"]; len(got) != 2 || got[0] != (event.FacetCount{Value: "web-2", Count: 2}) {
		t.Errorf("host counts = %v", got)
	}
	if got := counts["tag:route"]; len(got) != 1 || got[0].Count != 3 {
		t.Errorf("tag counts = %v", got)
	}

	var list []struct {
		UUID string `json:"uuid"`
	}
	text := callTool(t, session, "events_list", map[string]any{"host": []string{"web-1"}, "tag": []string{"route:/cart"}})
	if err := json.Unmarshal([]byte(text), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].UUID != web1 {
		t.Errorf("events_list host=web-1 = %s", text)
	}
	text = callTool(t, session, "events_list", map[string]any{"exclude_host": []string{"web-2"}})
	if err := json.Unmarshal([]byte(text), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].UUID != web1 {
		t.Errorf("events_list exclude web-2 = %s", text)
	}
}
