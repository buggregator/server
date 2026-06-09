package sentry

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestEnrichSourceContext_BrowserFrame(t *testing.T) {
	// A SvelteKit/browser frame: only filename(URL) + lineno, no source.
	payload := []byte(`{
		"event_id":"abc",
		"platform":"javascript",
		"exception":{"values":[{
			"type":"Error",
			"value":"boom",
			"stacktrace":{"frames":[
				{"filename":"https://app.test/src/routes/+page.svelte","function":"?","in_app":true,"lineno":3,"colno":10}
			]}
		}]}
	}`)

	src := "line1\nline2\nline3-THROW\nline4\nline5"
	fetch := func(url string) (string, bool) {
		if url == "https://app.test/src/routes/+page.svelte" {
			return src, true
		}
		return "", false
	}

	out, changed := enrichSourceContext(payload, fetch)
	if !changed {
		t.Fatal("expected payload to change")
	}

	frame := firstFrame(t, out)
	if got := frame["context_line"]; got != "line3-THROW" {
		t.Fatalf("context_line = %v, want line3-THROW", got)
	}
	pre := toStrings(frame["pre_context"])
	if len(pre) != 2 || pre[0] != "line1" || pre[1] != "line2" {
		t.Fatalf("pre_context = %v, want [line1 line2]", pre)
	}
	post := toStrings(frame["post_context"])
	if len(post) != 2 || post[0] != "line4" || post[1] != "line5" {
		t.Fatalf("post_context = %v, want [line4 line5]", post)
	}
}

func TestEnrichSourceContext_SkipsNonURLAndExistingSource(t *testing.T) {
	// PHP-style frame (filesystem path) and a frame that already has source.
	payload := []byte(`{
		"exception":{"values":[{"stacktrace":{"frames":[
			{"filename":"/var/www/app/Foo.php","lineno":7},
			{"filename":"https://app.test/a.js","lineno":1,"context_line":"already here"}
		]}}]}
	}`)

	called := false
	fetch := func(string) (string, bool) { called = true; return "x", true }

	_, changed := enrichSourceContext(payload, fetch)
	if changed {
		t.Fatal("should not change: no eligible frames")
	}
	if called {
		t.Fatal("fetch must not be called for filesystem paths or framed-with-source")
	}
}

func TestEnrichSourceContext_FetchFailureIsBestEffort(t *testing.T) {
	payload := []byte(`{"exception":{"values":[{"stacktrace":{"frames":[
		{"filename":"https://app.test/a.js","lineno":1}
	]}}]}}`)

	fetch := func(string) (string, bool) { return "", false }

	out, changed := enrichSourceContext(payload, fetch)
	if changed {
		t.Fatal("fetch failed, payload must be unchanged")
	}
	frame := firstFrame(t, out)
	if _, ok := frame["context_line"]; ok {
		t.Fatal("no context_line should be added on fetch failure")
	}
}

func TestEnrichSourceContext_TopLevelAndThreads(t *testing.T) {
	src := "a\nb\nc"
	fetch := func(string) (string, bool) { return src, true }

	for name, payload := range map[string]string{
		"top-level": `{"stacktrace":{"frames":[{"filename":"https://x/y.js","lineno":2}]}}`,
		"threads":   `{"threads":{"values":[{"stacktrace":{"frames":[{"filename":"https://x/y.js","lineno":2}]}}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, changed := enrichSourceContext([]byte(payload), fetch)
			if !changed {
				t.Fatalf("%s: expected enrichment", name)
			}
		})
	}
}

func TestSourceMap_OriginalPosition(t *testing.T) {
	// generated line 1 -> original line 1; generated line 2 -> original line 3.
	sm := &sourceMap{
		Mappings:       "AAAA;AAEA",
		Sources:        []string{"original.js"},
		SourcesContent: []string{"// header\n// header2\nthrow new Error(\"boom\");\n"},
	}
	srcIdx, origLine, _, ok := sm.originalPosition(2, 0)
	if !ok {
		t.Fatal("expected a mapping")
	}
	if srcIdx != 0 || origLine != 2 {
		t.Fatalf("got srcIdx=%d origLine=%d, want 0 and 2", srcIdx, origLine)
	}
	lines, _ := sm.originalSourceLines(srcIdx)
	if lines[origLine] != `throw new Error("boom");` {
		t.Fatalf("original line = %q", lines[origLine])
	}
}

func TestEnrichFrame_UsesOriginalSourceViaSourceMap(t *testing.T) {
	smJSON := `{"version":3,"sources":["original.js"],` +
		`"sourcesContent":["// header\n// header2\nthrow new Error(\"boom\");\n"],` +
		`"mappings":"AAAA;AAEA","names":[]}`
	b64 := base64.StdEncoding.EncodeToString([]byte(smJSON))
	transformed := "var x = 1;\nthrow new Error(\"boom\");\n" +
		"//# sourceMappingURL=data:application/json;base64," + b64

	fetch := func(string) (string, bool) { return transformed, true }

	// frame points at the GENERATED line 2; we expect the ORIGINAL (line 3) back.
	payload := []byte(`{"exception":{"values":[{"stacktrace":{"frames":[
		{"filename":"https://app/x.js","lineno":2,"colno":1}
	]}}]}}`)

	out, changed := enrichSourceContext(payload, fetch)
	if !changed {
		t.Fatal("expected enrichment")
	}
	frame := firstFrame(t, out)
	if got := frame["context_line"]; got != `throw new Error("boom");` {
		t.Fatalf("context_line = %v, want original throw line", got)
	}
	if got, _ := frame["lineno"].(float64); got != 3 {
		t.Fatalf("lineno = %v, want 3 (original)", frame["lineno"])
	}
	pre := toStrings(frame["pre_context"])
	if len(pre) != 2 || pre[0] != "// header" || pre[1] != "// header2" {
		t.Fatalf("pre_context = %v, want original header lines", pre)
	}
}

func firstFrame(t *testing.T, payload json.RawMessage) map[string]any {
	t.Helper()
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	exc := root["exception"].(map[string]any)
	vals := exc["values"].([]any)
	st := vals[0].(map[string]any)["stacktrace"].(map[string]any)
	frames := st["frames"].([]any)
	return frames[0].(map[string]any)
}

func toStrings(v any) []string {
	s, _ := v.([]any)
	out := make([]string, len(s))
	for i, e := range s {
		out[i], _ = e.(string)
	}
	return out
}
