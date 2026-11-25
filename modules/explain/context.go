package explain

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// Context kinds: an event the server reads itself, or a page the client
// describes in its own text.
const (
	KindEvent = "event"
	KindPage  = "page"
)

// Sizes of what goes to the agents. h2a takes an item's text up to
// itemTextBytes with its frame; the node may set a lower limit.
const (
	itemTextBytes = 8000
	maxEventBytes = 7000 // the event's own text, before the question
	maxPageBytes  = 4096 // the page text a client sends
	maxTitleBytes = 200
	maxKeyBytes   = 512
	topRows       = 15 // profiler functions, inspector segments
	blobBytes     = 2000
)

// Context is what a conversation is about.
type Context struct {
	Kind    string `json:"kind"`
	Key     string `json:"key"`
	Title   string `json:"title,omitempty"`
	Text    string `json:"text,omitempty"`
	Project string `json:"project,omitempty"`
}

var errContextTooLong = errors.New("page text is longer than 4096 bytes")

// check validates a context from a client; the page text is refused, not
// cut, when too long: the client knows best what to leave out.
func (c *Context) check() error {
	c.Kind = strings.TrimSpace(c.Kind)
	c.Key = strings.TrimSpace(c.Key)
	c.Title = oneLine(c.Title, maxTitleBytes)
	c.Project = strings.TrimSpace(c.Project)
	switch {
	case c.Kind != KindEvent && c.Kind != KindPage:
		return errors.New(`context.kind is "event" or "page"`)
	case c.Key == "":
		return errors.New("context.key is required")
	case len(c.Key) > maxKeyBytes:
		return errors.New("context.key is longer than 512 bytes")
	case c.Kind == KindPage && len(c.Text) > maxPageBytes:
		return errContextTooLong
	}
	if c.Kind == KindEvent {
		c.Text = "" // the server reads the event itself
	}
	return nil
}

// pointer tells the agents where to look further; never a token.
type pointer struct {
	MCP bool // Buggregator's MCP is on and read-only
}

// contextText is the first message of a conversation: an intro, the
// person's question, the context's own text (scrubbed, cut to fit) and the
// pointers. limit is the node's text limit (0: none given).
func contextText(c Context, ev *event.Event, question string, p pointer, limit int) (string, error) {
	if limit <= 0 || limit > itemTextBytes {
		limit = itemTextBytes
	}
	var intro, body string
	project := c.Project
	switch c.Kind {
	case KindEvent:
		b, err := eventBody(*ev)
		if err != nil {
			return "", err
		}
		body = cut(b, maxEventBytes)
		project = ev.Project
		if ev.Type == "sentry" {
			intro = sentryIntro
		} else if question == "" {
			intro = fmt.Sprintf("Look at this %s event caught by Buggregator (a debugging tool) and say what stands out: problems, likely causes, what to check.", ev.Type)
		} else {
			intro = fmt.Sprintf("A %s event caught by Buggregator (a debugging tool); the person asks about it.", ev.Type)
		}
	default:
		body = c.Text
		if question == "" {
			intro = "The person is looking at this page of Buggregator (a debugging tool). Say what stands out on it: problems, likely causes, what to check."
		} else {
			intro = "The person is looking at this page of Buggregator (a debugging tool) and asks about it."
		}
	}

	// Everything the person or the page wrote is masked, not only the body.
	var head strings.Builder
	head.WriteString(intro + "\n")
	if question != "" {
		head.WriteString("Question: " + scrub(question) + "\n")
	}
	if c.Title != "" {
		head.WriteString("Page: " + scrub(c.Title) + "\n")
	}
	head.WriteString("\n")

	var tail strings.Builder
	tail.WriteString("\n")
	if c.Kind == KindEvent {
		fmt.Fprintf(&tail, "Event: %s (%s, project %s)\n", ev.UUID, ev.Type, project)
	} else {
		tail.WriteString("Page: " + scrub(c.Key) + "\n")
	}
	if p.MCP {
		tail.WriteString("Buggregator data is available via the `buggregator` MCP connection (read-only)")
		if c.Kind == KindEvent {
			tail.WriteString(": event " + ev.UUID)
			if project != "" {
				tail.WriteString(", project " + project)
			}
		} else if project != "" {
			tail.WriteString(": project " + project)
		}
		tail.WriteString(".\n")
	}

	room := limit - head.Len() - tail.Len()
	return head.String() + cut(scrub(body), room) + tail.String(), nil
}

// eventBody is an event's own text by type; any other type gets its JSON.
func eventBody(ev event.Event) (string, error) {
	switch ev.Type {
	case "sentry":
		return sentryBody(ev.Payload)
	case "monolog":
		return monologBody(ev.Payload), nil
	case "http-dump":
		return httpDumpBody(ev.Payload), nil
	case "var-dump":
		return varDumpBody(ev.Payload), nil
	case "profiler":
		return profilerBody(ev.Payload), nil
	case "inspector":
		return inspectorBody(ev.Payload), nil
	}
	return genericBody(ev), nil
}

func genericBody(ev event.Event) string {
	return "Type: " + ev.Type + "\nPayload (JSON):\n" + cut(compact(ev.Payload), maxEventBytes-200) + "\n"
}

func monologBody(payload json.RawMessage) string {
	var r struct {
		Message   string          `json:"message"`
		Channel   string          `json:"channel"`
		LevelName string          `json:"level_name"`
		Datetime  string          `json:"datetime"`
		Context   json.RawMessage `json:"context"`
		Extra     json.RawMessage `json:"extra"`
	}
	if json.Unmarshal(payload, &r) != nil {
		return "Log record (JSON):\n" + cut(compact(payload), maxEventBytes-200) + "\n"
	}
	var b strings.Builder
	writeField(&b, "Log", strings.TrimSpace(r.Channel+"."+r.LevelName))
	writeField(&b, "Time", r.Datetime)
	b.WriteString("Message: " + cut(r.Message, blobBytes) + "\n")
	writeBlob(&b, "Context", r.Context)
	writeBlob(&b, "Extra", r.Extra)
	return b.String()
}

func httpDumpBody(payload json.RawMessage) string {
	var d struct {
		Host       string   `json:"host"`
		Scheme     string   `json:"scheme"`
		DurationMs *float64 `json:"duration_ms"`
		Error      string   `json:"error"`
		Request    struct {
			Method  string              `json:"method"`
			URI     string              `json:"uri"`
			Headers map[string][]string `json:"headers"`
			Body    string              `json:"body"`
		} `json:"request"`
		Response *struct {
			StatusCode int                 `json:"status_code"`
			Headers    map[string][]string `json:"headers"`
			Body       string              `json:"body"`
		} `json:"response"`
	}
	if json.Unmarshal(payload, &d) != nil {
		return "HTTP request (JSON):\n" + cut(compact(payload), maxEventBytes-200) + "\n"
	}
	var b strings.Builder
	b.WriteString("Request: " + strings.TrimSpace(d.Request.Method+" "+d.Host+d.Request.URI) + "\n")
	writeHeaders(&b, d.Request.Headers)
	if body := strings.TrimSpace(d.Request.Body); body != "" {
		b.WriteString("Request body:\n" + cut(body, blobBytes) + "\n")
	}
	if d.Response != nil {
		fmt.Fprintf(&b, "\nResponse: %d", d.Response.StatusCode)
		if d.DurationMs != nil {
			fmt.Fprintf(&b, " in %.0f ms", *d.DurationMs)
		}
		b.WriteString("\n")
		writeHeaders(&b, d.Response.Headers)
		if body := strings.TrimSpace(d.Response.Body); body != "" {
			b.WriteString("Response body:\n" + cut(body, blobBytes) + "\n")
		}
	}
	writeField(&b, "Proxy error", d.Error)
	return b.String()
}

// writeHeaders lists headers sorted, one per line; secrets go in scrub.
func writeHeaders(b *strings.Builder, h map[string][]string) {
	if len(h) == 0 {
		return
	}
	names := make([]string, 0, len(h))
	for k := range h {
		names = append(names, k)
	}
	sort.Strings(names)
	b.WriteString("Headers:\n")
	for _, k := range names {
		b.WriteString("  " + k + ": " + oneLine(strings.Join(h[k], ", "), 300) + "\n")
	}
}

var htmlTag = regexp.MustCompile(`<[^>]*>`)

func varDumpBody(payload json.RawMessage) string {
	var d struct {
		Payload struct {
			Type     string `json:"type"`
			Value    string `json:"value"`
			Label    string `json:"label"`
			Language string `json:"language"`
		} `json:"payload"`
		Context json.RawMessage `json:"context"`
	}
	if json.Unmarshal(payload, &d) != nil {
		return "Dump (JSON):\n" + cut(compact(payload), maxEventBytes-200) + "\n"
	}
	value := d.Payload.Value
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		value = html.UnescapeString(htmlTag.ReplaceAllString(value, ""))
	}
	var b strings.Builder
	writeField(&b, "Dump", strings.TrimSpace(d.Payload.Label+" ("+d.Payload.Type+")"))
	writeField(&b, "Language", d.Payload.Language)
	b.WriteString("Value:\n" + cut(strings.TrimSpace(value), 4000) + "\n")
	writeBlob(&b, "Source", d.Context)
	return b.String()
}

func profilerBody(payload json.RawMessage) string {
	type metrics struct {
		WT  int64 `json:"wt"`
		CPU int64 `json:"cpu"`
		MU  int64 `json:"mu"`
		PMU int64 `json:"pmu"`
		CT  int64 `json:"ct"`
	}
	var d struct {
		AppName  string         `json:"app_name"`
		Hostname string         `json:"hostname"`
		Date     any            `json:"date"`
		Tags     map[string]any `json:"tags"`
		Peaks    metrics        `json:"peaks"`
		Edges    map[string]struct {
			Callee string  `json:"callee"`
			Cost   metrics `json:"cost"`
			Diff   struct {
				WT int64 `json:"d_wt"`
				MU int64 `json:"d_mu"`
			} `json:"diff"`
		} `json:"edges"`
	}
	if json.Unmarshal(payload, &d) != nil {
		return genericBody(event.Event{Type: "profiler", Payload: payload})
	}
	var b strings.Builder
	writeField(&b, "App", d.AppName)
	writeField(&b, "Host", d.Hostname)
	if len(d.Tags) > 0 {
		tags, _ := json.Marshal(d.Tags)
		writeField(&b, "Tags", string(tags))
	}
	fmt.Fprintf(&b, "Totals: wall %s, cpu %s, memory %s, peak memory %s\n",
		ms(d.Peaks.WT), ms(d.Peaks.CPU), kb(d.Peaks.MU), kb(d.Peaks.PMU))

	type fn struct {
		name          string
		excl, incl, n int64
		mem           int64
	}
	byName := map[string]*fn{}
	for _, e := range d.Edges {
		f := byName[e.Callee]
		if f == nil {
			f = &fn{name: e.Callee}
			byName[e.Callee] = f
		}
		f.excl += e.Diff.WT
		f.incl += e.Cost.WT
		f.n += e.Cost.CT
		f.mem += e.Diff.MU
	}
	list := make([]*fn, 0, len(byName))
	for _, f := range byName {
		list = append(list, f)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].excl != list[j].excl {
			return list[i].excl > list[j].excl
		}
		return list[i].name < list[j].name
	})
	if len(list) > topRows {
		list = list[:topRows]
	}
	if len(list) > 0 {
		b.WriteString("Slowest functions by own (exclusive) wall time:\n")
		for _, f := range list {
			fmt.Fprintf(&b, "  %s — own %s, with callees %s, calls %d, own memory %s\n", f.name, ms(f.excl), ms(f.incl), f.n, kb(f.mem))
		}
	}
	return b.String()
}

func inspectorBody(payload json.RawMessage) string {
	var items []map[string]any
	if json.Unmarshal(payload, &items) != nil {
		return genericBody(event.Event{Type: "inspector", Payload: payload})
	}
	var b strings.Builder
	type seg struct {
		label string
		dur   float64
	}
	var segs []seg
	for _, it := range items {
		model, _ := it["model"].(string)
		switch model {
		case "transaction":
			fmt.Fprintf(&b, "Transaction: %v (%v), result %v, %v ms\n", it["name"], it["type"], it["result"], it["duration"])
			if host, ok := it["host"].(map[string]any); ok {
				writeField(&b, "Host", fmt.Sprint(host["hostname"]))
			}
		case "segment":
			d, _ := it["duration"].(float64)
			segs = append(segs, seg{strings.TrimSpace(fmt.Sprintf("%v: %v", it["type"], it["label"])), d})
		case "error":
			fmt.Fprintf(&b, "Error: %v: %v at %v:%v\n", it["class"], oneLine(fmt.Sprint(it["message"]), 300), it["file"], it["line"])
		}
	}
	sort.SliceStable(segs, func(i, j int) bool { return segs[i].dur > segs[j].dur })
	if len(segs) > 0 {
		fmt.Fprintf(&b, "Slowest segments (%d in all):\n", len(segs))
		for i, s := range segs {
			if i == topRows {
				break
			}
			fmt.Fprintf(&b, "  %.1f ms — %s\n", s.dur, oneLine(s.label, 200))
		}
	}
	if b.Len() == 0 {
		return genericBody(event.Event{Type: "inspector", Payload: payload})
	}
	return b.String()
}

func writeBlob(b *strings.Builder, name string, raw json.RawMessage) {
	s := compact(raw)
	if s == "" || s == "null" || s == "{}" || s == "[]" {
		return
	}
	b.WriteString(name + ": " + cut(s, blobBytes) + "\n")
}

func compact(raw json.RawMessage) string {
	var buf strings.Builder
	if len(raw) == 0 {
		return ""
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return strings.TrimSpace(buf.String())
}

func ms(us int64) string { return fmt.Sprintf("%.1f ms", float64(us)/1000) }
func kb(b int64) string  { return fmt.Sprintf("%.0f KB", float64(b)/1024) }
