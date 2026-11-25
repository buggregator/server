package explain

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/buggregator/server/v2/modules/sentry"
)

// Limits of the text sent to the agents.
const (
	maxFrames        = 8  // app frames listed per exception
	contextFrames    = 3  // frames that also show their source lines
	maxBreadcrumbs   = 10 // the last breadcrumbs before the error
	defaultTextBytes = 8000
)

type frame struct {
	Filename    string   `json:"filename"`
	AbsPath     string   `json:"abs_path"`
	Function    string   `json:"function"`
	Module      string   `json:"module"`
	Lineno      int      `json:"lineno"`
	InApp       *bool    `json:"in_app"`
	ContextLine string   `json:"context_line"`
	PreContext  []string `json:"pre_context"`
	PostContext []string `json:"post_context"`
}

type stacktrace struct {
	Frames []frame `json:"frames"`
}

// Prompt builds the plain text the agents get for a Sentry error event,
// at most limit bytes (0: defaultTextBytes). question, when set, is the
// person's own words, put first.
func Prompt(payload json.RawMessage, question string, limit int) (string, error) {
	if limit <= 0 {
		limit = defaultTextBytes
	}
	var ev sentry.ErrorEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return "", fmt.Errorf("not a Sentry event: %w", err)
	}

	var b strings.Builder
	b.WriteString("Explain this error caught by Buggregator: what happened, the likely cause, and how to fix it.\n")
	if q := strings.TrimSpace(question); q != "" {
		b.WriteString("Question: " + q + "\n")
	}
	b.WriteString("\n")

	exceptions := []sentry.ExceptionValue(nil)
	if ev.Exception != nil {
		exceptions = ev.Exception.Values
	}
	// Sentry lists the chain oldest first; the error itself is the last one.
	for i := len(exceptions) - 1; i >= 0; i-- {
		ex := exceptions[i]
		if i == len(exceptions)-1 {
			b.WriteString("Error: ")
		} else {
			b.WriteString("\nCaused by: ")
		}
		b.WriteString(strings.TrimSpace(ex.Type + ": " + ex.Value))
		b.WriteString("\n")
		writeStack(&b, ex.Stacktrace)
	}
	if len(exceptions) == 0 {
		msg := ev.Message.String()
		if msg == "" && ev.LogEntry != nil {
			msg = ev.LogEntry.Message
		}
		b.WriteString("Message: " + msg + "\n")
	}

	b.WriteString("\n")
	writeField(&b, "Level", ev.Level)
	writeField(&b, "Platform", ev.Platform)
	writeField(&b, "Environment", ev.Environment)
	writeField(&b, "Release", ev.Release)
	writeField(&b, "Transaction", ev.Transaction)
	writeField(&b, "Server", ev.ServerName)
	if req := request(ev.Request); req != "" {
		writeField(&b, "Request", req)
	}
	if tags := tags(ev.Tags); tags != "" {
		writeField(&b, "Tags", tags)
	}
	if ev.Breadcrumbs != nil && len(ev.Breadcrumbs.Values) > 0 {
		crumbs := ev.Breadcrumbs.Values
		if len(crumbs) > maxBreadcrumbs {
			crumbs = crumbs[len(crumbs)-maxBreadcrumbs:]
		}
		b.WriteString("\nLast steps before the error:\n")
		for _, c := range crumbs {
			line := strings.TrimSpace(strings.Join(nonEmpty(c.Category, c.Message), " "))
			if line != "" {
				b.WriteString("  - " + oneLine(line, 200) + "\n")
			}
		}
	}
	return cut(b.String(), limit), nil
}

func writeStack(b *strings.Builder, raw json.RawMessage) {
	var st stacktrace
	if len(raw) == 0 || json.Unmarshal(raw, &st) != nil || len(st.Frames) == 0 {
		return
	}
	frames := st.Frames
	var app []frame
	for _, f := range frames {
		if f.InApp != nil && *f.InApp {
			app = append(app, f)
		}
	}
	if len(app) > 0 {
		frames = app
	}
	b.WriteString("Stack (most recent call first):\n")
	shown := 0
	for i := len(frames) - 1; i >= 0 && shown < maxFrames; i-- {
		f := frames[i]
		file := f.Filename
		if file == "" {
			file = f.AbsPath
		}
		if file == "" {
			file = f.Module
		}
		fmt.Fprintf(b, "  at %s:%d", file, f.Lineno)
		if f.Function != "" {
			b.WriteString(" in " + f.Function)
		}
		b.WriteString("\n")
		if shown < contextFrames && f.ContextLine != "" {
			start := f.Lineno - len(f.PreContext)
			for j, l := range f.PreContext {
				fmt.Fprintf(b, "      %d | %s\n", start+j, codeLine(l))
			}
			fmt.Fprintf(b, "    > %d | %s\n", f.Lineno, codeLine(f.ContextLine))
			for j, l := range f.PostContext {
				fmt.Fprintf(b, "      %d | %s\n", f.Lineno+1+j, codeLine(l))
			}
		}
		shown++
	}
}

// request gives the method and URL only: headers, cookies and the body stay home.
func request(raw json.RawMessage) string {
	var r struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &r) != nil {
		return ""
	}
	return strings.TrimSpace(r.Method + " " + r.URL)
}

// tags reads both shapes SDKs send: an object or a list of [key, value].
func tags(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var parts []string
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil {
		for k, v := range obj {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	} else {
		var pairs [][]any
		if json.Unmarshal(raw, &pairs) == nil {
			for _, p := range pairs {
				if len(p) == 2 {
					parts = append(parts, fmt.Sprintf("%v=%v", p[0], p[1]))
				}
			}
		}
	}
	sort.Strings(parts)
	return oneLine(strings.Join(parts, ", "), 400)
}

func writeField(b *strings.Builder, name, value string) {
	if value = strings.TrimSpace(value); value != "" {
		b.WriteString(name + ": " + oneLine(value, 300) + "\n")
	}
}

func nonEmpty(s ...string) []string {
	var out []string
	for _, v := range s {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// codeLine keeps a source line's indentation (it matters in Python, YAML…).
func codeLine(s string) string {
	s = strings.TrimRight(strings.NewReplacer("\r", "", "\n", " ", "\t", "    ").Replace(s), " ")
	if len(s) > 160 {
		s = truncate(s, 159) + "…"
	}
	return s
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > max {
		s = truncate(s, max-1) + "…"
	}
	return s
}

// cut keeps the text within limit bytes, on a rune boundary.
func cut(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	const tail = "\n[cut to fit]"
	return truncate(s, limit-len(tail)) + tail
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && (s[n]&0xC0) == 0x80 {
		n--
	}
	return s[:n]
}
