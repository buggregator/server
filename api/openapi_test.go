package api

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The routes the server registers, read from the source, must match the
// operations in openapi.yaml: a new route needs its entry here, and an entry
// for a removed route fails too. Ingestion paths (x-ingestion) are matched by
// the event handlers, not by the mux, and are skipped.

var (
	// "GET /api/events" in mux.Handle / HandleFunc / protect / h(...) calls.
	literalRoute = regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE) (/[^"\s?]*)"`)
	// "GET "+prefix+"/{uuid}" in registerAttachmentsForModule.
	prefixRoute = regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE) "\s*\+\s*prefix\s*\+\s*"([^"]*)"`)
	// registerAttachmentsForModule(..., "/api/smtp/attachments", ...)
	attachmentPrefix = regexp.MustCompile(`registerAttachmentsForModule\([^)]*"(/api/[^"]+)"`)
	// "POST /api/xdebug/sessions/{id}/"+route over controlRoutes.
	controlRoute = regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE) (/[^"]*/)"\s*\+\s*route`)
	controlMap   = regexp.MustCompile(`(?s)var controlRoutes = map\[string\]string\{(.*?)\n\}`)
	mapKey       = regexp.MustCompile(`"([a-z-]+)":`)
	// Any other "METHOD ..."+expr is a route this test cannot read yet.
	// SQL ("DELETE FROM " + table) starts with no slash and is skipped.
	unknownDynamic = regexp.MustCompile(`"(GET|POST|PUT|PATCH|DELETE) /[^"]*"\s*\+`)
	// A literal followed by + is only the start of a dynamic route.
	continued = regexp.MustCompile(`^\s*\+`)
)

// documented are the paths of the HTTP API; other literal routes in the source
// (an SDK's request in a log message, demo seed data) are not routes.
func documented(path string) bool {
	for _, p := range []string{"/api/", "/auth/", "/metrics", "/connection/websocket", "/ws"} {
		if path == strings.TrimSuffix(p, "/") || strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func sourceRoutes(t *testing.T) map[string]bool {
	t.Helper()
	routes := map[string]bool{}
	var prefixSuffixes []string
	var attachmentPrefixes []string
	var controlBases []string
	var controlNames []string

	for _, dir := range []string{"../internal", "../modules"} {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			src := string(raw)
			for _, line := range strings.Split(src, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "//") {
					continue
				}
				for _, idx := range literalRoute.FindAllStringSubmatchIndex(line, -1) {
					method, path := line[idx[2]:idx[3]], line[idx[4]:idx[5]]
					if documented(path) && !continued.MatchString(line[idx[1]:]) {
						routes[method+" "+path] = true
					}
				}
				handled := false
				for _, m := range prefixRoute.FindAllStringSubmatch(line, -1) {
					prefixSuffixes = append(prefixSuffixes, m[1]+" "+m[2])
					handled = true
				}
				for _, m := range attachmentPrefix.FindAllStringSubmatch(line, -1) {
					attachmentPrefixes = append(attachmentPrefixes, m[1])
				}
				for _, m := range controlRoute.FindAllStringSubmatch(line, -1) {
					controlBases = append(controlBases, m[1]+" "+m[2])
					handled = true
				}
				if !handled && unknownDynamic.MatchString(line) {
					t.Errorf("%s: route built from an expression this test cannot read; extend api/openapi_test.go:\n%s", path, strings.TrimSpace(line))
				}
			}
			if m := controlMap.FindStringSubmatch(src); m != nil {
				for _, k := range mapKey.FindAllStringSubmatch(m[1], -1) {
					controlNames = append(controlNames, k[1])
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(prefixSuffixes) > 0 && len(attachmentPrefixes) == 0 {
		t.Fatal("attachment routes found but no registerAttachmentsForModule prefixes")
	}
	for _, ms := range prefixSuffixes {
		method, suffix, _ := strings.Cut(ms, " ")
		for _, p := range attachmentPrefixes {
			routes[method+" "+p+suffix] = true
		}
	}
	if len(controlBases) > 0 && len(controlNames) == 0 {
		t.Fatal("xdebug control routes found but no controlRoutes keys")
	}
	for _, base := range controlBases {
		for _, name := range controlNames {
			routes[base+name] = true
		}
	}
	return routes
}

type operation struct {
	Ingestion bool `yaml:"x-ingestion"`
}

func specRoutes(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("openapi.yaml: %v", err)
	}
	routes := map[string]bool{}
	for path, item := range spec.Paths {
		for method, node := range item {
			switch method {
			case "get", "post", "put", "patch", "delete":
			default:
				continue // parameters, summary, ...
			}
			var op operation
			if err := node.Decode(&op); err != nil {
				t.Fatalf("%s %s: %v", method, path, err)
			}
			if op.Ingestion {
				continue
			}
			routes[strings.ToUpper(method)+" "+path] = true
		}
	}
	return routes
}

func TestOpenAPIMatchesRoutes(t *testing.T) {
	code := sourceRoutes(t)
	spec := specRoutes(t)
	if len(code) < 50 {
		t.Fatalf("read only %d routes from the source; the patterns no longer match the code", len(code))
	}

	var missing, stale []string
	for r := range code {
		if !spec[r] {
			missing = append(missing, r)
		}
	}
	for r := range spec {
		if !code[r] {
			stale = append(stale, r)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	for _, r := range missing {
		t.Errorf("route %s is not in api/openapi.yaml", r)
	}
	for _, r := range stale {
		t.Errorf("api/openapi.yaml has %s, but the server registers no such route", r)
	}
}
