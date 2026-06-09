package sentry

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

// Source context resolution for browser/JS stack frames.
//
// JavaScript SDKs (e.g. sentry.javascript.sveltekit) send error frames that
// carry only `filename` (a URL), `lineno` and `colno` — no source code. The
// frontend cannot fetch that source itself because the dev server is a
// different origin and CORS blocks it. So we resolve the code here, server-side,
// where there is no CORS restriction: fetch the file the frame points at and
// attach pre_context / context_line / post_context. This is what makes
// SvelteKit (and any Vite/webpack dev-server) JS stack traces actually show
// code in Buggregator.
//
// It is strictly best-effort and self-limiting:
//   - only http(s) `filename`s are touched, which naturally scopes this to
//     browser/JS frames (PHP/Python frames carry filesystem paths, not URLs);
//   - frames that already include source are left alone;
//   - any fetch error leaves the frame untouched.

const (
	sourceContextLines = 5            // lines of context kept before and after
	sourceFetchTimeout = 3 * time.Second
	sourceMaxBytes     = 2 << 20      // 2 MiB cap per fetched file
)

// sourceFetcher returns the text of a remote source file and whether it was
// retrieved. It is an injection point for tests.
type sourceFetcher func(url string) (string, bool)

var defaultSourceClient = &http.Client{
	Timeout: sourceFetchTimeout,
	Transport: &http.Transport{
		// Local dev servers (Vite behind ddev/https) use self-signed certs.
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
}

func httpSourceFetcher(url string) (string, bool) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", false
	}
	resp, err := defaultSourceClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, sourceMaxBytes))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// enrichSourceContext walks every stacktrace in a Sentry error payload and
// attaches source code to frames that reference a fetchable URL but carry none.
// It returns the (possibly rewritten) payload and whether anything changed.
func enrichSourceContext(payload json.RawMessage, fetch sourceFetcher) (json.RawMessage, bool) {
	if fetch == nil {
		fetch = httpSourceFetcher
	}

	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return payload, false
	}

	cache := map[string][]string{} // url -> lines (per-event, dedupes refetches)
	missing := map[string]bool{}   // url -> fetch already failed
	changed := false

	visit := func(st any) {
		if enrichStacktrace(st, fetch, cache, missing) {
			changed = true
		}
	}

	if exc, ok := root["exception"].(map[string]any); ok {
		for _, v := range asAnySlice(exc["values"]) {
			if vm, ok := v.(map[string]any); ok {
				visit(vm["stacktrace"])
			}
		}
	}
	if th, ok := root["threads"].(map[string]any); ok {
		for _, v := range asAnySlice(th["values"]) {
			if vm, ok := v.(map[string]any); ok {
				visit(vm["stacktrace"])
			}
		}
	}
	visit(root["stacktrace"])

	if !changed {
		return payload, false
	}

	out, err := json.Marshal(root)
	if err != nil {
		return payload, false
	}
	return out, true
}

func asAnySlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func enrichStacktrace(st any, fetch sourceFetcher, cache map[string][]string, missing map[string]bool) bool {
	stm, ok := st.(map[string]any)
	if !ok {
		return false
	}
	changed := false
	for _, f := range asAnySlice(stm["frames"]) {
		if fm, ok := f.(map[string]any); ok {
			if enrichFrame(fm, fetch, cache, missing) {
				changed = true
			}
		}
	}
	return changed
}

func enrichFrame(fm map[string]any, fetch sourceFetcher, cache map[string][]string, missing map[string]bool) bool {
	// Skip frames that already carry source.
	if s, ok := fm["context_line"].(string); ok && s != "" {
		return false
	}

	filename, _ := fm["filename"].(string)
	if !strings.HasPrefix(filename, "http://") && !strings.HasPrefix(filename, "https://") {
		return false
	}

	lineno, ok := toLineNumber(fm["lineno"])
	if !ok || lineno < 1 {
		return false
	}

	lines, ok := cache[filename]
	if !ok {
		if missing[filename] {
			return false
		}
		text, fetched := fetch(filename)
		if !fetched {
			missing[filename] = true
			return false
		}
		lines = strings.Split(text, "\n")
		cache[filename] = lines
	}

	if lineno > len(lines) {
		return false
	}

	idx := lineno - 1
	fm["context_line"] = lines[idx]

	pre := make([]any, 0, sourceContextLines)
	for i := max(0, idx-sourceContextLines); i < idx; i++ {
		pre = append(pre, lines[i])
	}
	post := make([]any, 0, sourceContextLines)
	for i := idx + 1; i < min(len(lines), idx+1+sourceContextLines); i++ {
		post = append(post, lines[i])
	}
	fm["pre_context"] = pre
	fm["post_context"] = post
	return true
}

func toLineNumber(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case int:
		return n, true
	}
	return 0, false
}
