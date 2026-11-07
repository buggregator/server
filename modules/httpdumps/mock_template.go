package httpdumps

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"text/template/parse"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// Mock response templates (#4493): with Template on, a rule's body and header
// values are Go text/template sources. Helpers read the request and make fake
// data (gofakeit). MockHelpers documents every helper; the MCP tools, the
// /api/mocks/helpers endpoint and the docs are built from it.

const (
	// MaxMockRepeat caps the loop items of one render (repeat, and range over
	// queryAll or body), so nested loops stay bounded even when they write
	// nothing. Other loops and recursion are refused at parse time.
	MaxMockRepeat = 10000
	// MaxMockWork caps the steps of one render: each loop item costs the
	// number of steps (actions, commands, text) in the loop's body.
	MaxMockWork = 1_000_000
	// MaxMockGenerated caps the bytes helpers make in one render (fake, json,
	// print…), written out or not.
	MaxMockGenerated = 4 << 20
	// maxFakeCount caps count-like fake parameters (words, length, count…).
	maxFakeCount = 1000
	// maxMockRequestBody is how much of the request body `body` and `bodyRaw` read.
	maxMockRequestBody = 1 << 20
)

// MockHelper documents one template helper.
type MockHelper struct {
	Name        string `json:"name"`
	Group       string `json:"group"` // request, data, output, fake
	Usage       string `json:"usage"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Result      string `json:"result"`
}

// MockHelpers is the reference of all template helpers, in reading order.
var MockHelpers = []MockHelper{
	{"path", "request", `path "name"`, `Value of a path parameter of the rule's path, as a string. "*" is the rest matched by a trailing *. Missing: "".`,
		`{"id": {{ path "id" }}}`, `rule /users/{id}, request /mock/users/42 → {"id": 42}`},
	{"query", "request", `query "name"`, `First value of a query-string parameter, as a string. Missing: "".`,
		`{"page": {{ query "page" | default "1" }}}`, `?page=3 → {"page": 3}; no page → {"page": 1}`},
	{"queryAll", "request", `queryAll "name"`, `All values of a query-string parameter, as a list of strings.`,
		`{"tags": {{ queryAll "tag" | json }}}`, `?tag=a&tag=b → {"tags": ["a","b"]}`},
	{"header", "request", `header "Name"`, `Value of a request header (name is case-insensitive). Missing: "".`,
		`{"request_id": {{ header "X-Request-Id" | json }}}`, `X-Request-Id: abc → {"request_id": "abc"}`},
	{"body", "request", `body "dot.path"`, `Value from the JSON request body by a dot path; list items by index (items.0.id). Returns the JSON value as is (string, number, bool, object, list); missing or no JSON body: nothing. Use | json to write it.`,
		`{"echo": {{ body "user.email" | json }}}`, `body {"user":{"email":"a@example.com"}} → {"echo": "a@example.com"}`},
	{"bodyRaw", "request", `bodyRaw`, `The request body as a string (first 1 MB).`,
		`{{ bodyRaw }}`, `echoes the body back`},
	{"method", "request", `method`, `Request method: GET, POST, …`,
		`{"method": {{ method | json }}}`, `{"method": "POST"}`},
	{"requestPath", "request", `requestPath`, `Request path without the /mock prefix.`,
		`{"path": {{ requestPath | json }}}`, `/mock/users/42 → {"path": "/users/42"}`},

	{"fake", "fake", `fake "name" [args…]`, `Fake data from gofakeit by generator name (case-insensitive). Arguments fill the generator's parameters in order. Strings need | json in JSON bodies; numbers and bools can be written as is. Objects (e.g. "person") need | json. Full list with parameters: GET /api/mocks/helpers.`,
		`{"name": {{ fake "name" | json }}, "age": {{ fake "number" 18 90 }}}`, `{"name": "Lavina Kuhn", "age": 37}`},
	{"pick", "data", `pick a b c…`, `One of the arguments at random.`,
		`{"status": {{ pick "paid" "pending" "failed" | json }}}`, `{"status": "pending"}`},
	{"repeat", "data", `repeat n`, `A list 0…n-1 for range: repeats a block n times. Inside, $i is the index. range works only over repeat, queryAll and body; all loop items in one response add up to at most 10000, and items × the steps in the loop body to at most 1000000. Helpers may make at most 4 MB per response. define, template and block are not supported.`,
		`[{{ range $i := repeat 3 }}{{ if $i }},{{ end }}{"n": {{ add $i 1 }}}{{ end }}]`, `[{"n": 1},{"n": 2},{"n": 3}]`},
	{"add", "data", `add a b`, `Sum of two integers (use int on request values first).`,
		`{{ add 2 3 }}`, `5`},
	{"int", "data", `int value`, `A request value as an integer: a string ("42", "4.9" → 4), a JSON number from body, a bool (1/0); missing or "" → 0; other text is an error. Request values are strings and body numbers are decimals, so convert them before repeat, add or comparing with a whole number.`,
		`{{ range $i := repeat (int (query "limit" | default "2")) }}{{ $i }}{{ end }} {{ if gt (int (body "qty")) 10 }}bulk{{ end }}`, `?limit=3 → 012; body {"qty": 12} → bulk`},
	{"now", "data", `now ["layout"]`, `Current time, UTC. Default RFC 3339; a Go time layout (e.g. "2006-01-02") or "unix" for seconds.`,
		`{"created_at": {{ now | json }}, "ts": {{ now "unix" }}}`, `{"created_at": "2026-11-07T10:00:00Z", "ts": 1794045600}`},

	{"json", "output", `json value`, `Writes a value as JSON: strings quoted and escaped, lists and objects encoded. Use it for every string in a JSON body.`,
		`{"name": {{ fake "name" | json }}}`, `{"name": "O'Neil \"Jr\""}`},
	{"default", "output", `default fallback value`, `value, or fallback when value is empty ("", missing). Pipe form: value | default fallback.`,
		`{{ query "limit" | default "20" }}`, `no limit → 20`},
	{"upper", "output", `upper s`, `Upper case.`, `{{ upper "abc" }}`, `ABC`},
	{"lower", "output", `lower s`, `Lower case.`, `{{ lower "ABC" }}`, `abc`},
}

// fakeExcluded are gofakeit generators that write files, run user patterns or
// make large binary output: not offered in templates.
var fakeExcluded = map[string]bool{
	"sql": true, "csv": true, "json": true, "xml": true, "fixed_width": true,
	"generate": true, "regex": true, "map": true, "template": true,
	"imagejpeg": true, "imagepng": true, "svg": true,
}

// MockFakeParam is one parameter of a fake generator.
type MockFakeParam struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Optional bool     `json:"optional"`
	Default  string   `json:"default,omitempty"`
	Options  []string `json:"options,omitempty"`
}

// MockFake documents one fake generator for `fake "name" …`.
type MockFake struct {
	Name        string          `json:"name"`
	Category    string          `json:"category"`
	Description string          `json:"description"`
	Example     string          `json:"example"`
	Output      string          `json:"output"`
	Params      []MockFakeParam `json:"params"`
}

// MockFakes lists the fake generators templates can use, by category then name.
func MockFakes() []MockFake {
	var out []MockFake
	for name, info := range gofakeit.FuncLookups {
		if !fakeAllowed(name, &info) {
			continue
		}
		f := MockFake{Name: name, Category: info.Category, Description: info.Description,
			Example: info.Example, Output: info.Output, Params: []MockFakeParam{}}
		for _, p := range info.Params {
			f.Params = append(f.Params, MockFakeParam{Name: p.Field, Type: p.Type, Optional: p.Optional,
				Default: p.Default, Options: p.Options})
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func fakeAllowed(name string, info *gofakeit.Info) bool {
	if fakeExcluded[name] {
		return false
	}
	for _, p := range info.Params {
		if p.Type == "[]Field" {
			return false
		}
	}
	return true
}

// mockRequest is what the request helpers read.
type mockRequest struct {
	Method string
	Path   string // without MockPrefix
	Params map[string]string
	Query  url.Values
	Header http.Header
	Body   []byte

	parsed  bool
	bodyVal any
}

// mockPathParams returns the {name} segments of pattern taken from path, and
// "*" for the rest matched by a trailing *.
func mockPathParams(pattern, path string) map[string]string {
	params := map[string]string{}
	ps, xs := splitPath(pattern), splitPath(path)
	for i, p := range ps {
		if p == "*" {
			if i < len(xs) {
				params["*"] = strings.Join(xs[i:], "/")
			} else {
				params["*"] = ""
			}
			break
		}
		if i < len(xs) && strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			params[p[1:len(p)-1]] = xs[i]
		}
	}
	return params
}

// renderState is one render: the request, the faker and the repeat budget.
type renderState struct {
	req       *mockRequest
	faker     *gofakeit.Faker
	repeats   int
	work      int
	generated int
	rawBody   *string
	// deadline ends loops and helpers of a slow render (zero = none);
	// timedOut records that it did.
	deadline time.Time
	timedOut bool
}

// MaxMockRenderTime bounds one render: what the step and byte budgets do not
// price (e.g. comparing two 1 MB strings) still stops in time.
var MaxMockRenderTime = 2 * time.Second

var errMockSlow = fmt.Errorf("template took longer than %v", MaxMockRenderTime)

func (s *renderState) late() bool {
	if !s.deadline.IsZero() && time.Now().After(s.deadline) {
		s.timedOut = true
	}
	return s.timedOut
}

// items iterates keys/values for a range, the way a slice or map would with
// decls range variables, and ends early past the deadline.
func (s *renderState) items(decls int, keys, vals []any) any {
	if decls > 1 {
		return iter.Seq2[any, any](func(yield func(any, any) bool) {
			for i := range vals {
				if s.late() || !yield(keys[i], vals[i]) {
					return
				}
			}
		})
	}
	return iter.Seq[any](func(yield func(any) bool) {
		for i := range vals {
			if s.late() || !yield(vals[i]) {
				return
			}
		}
	})
}

func indexed[T any](list []T) (keys, vals []any) {
	keys, vals = make([]any, len(list)), make([]any, len(list))
	for i, v := range list {
		keys[i], vals[i] = i, v
	}
	return keys, vals
}

// hiddenFuncs are bound in every template but not documented helpers: the
// range twins (see rangeFuncs) and builtins replaced to count their output.
var hiddenFuncs = map[string]bool{
	"rangeRepeat": true, "rangeQueryAll": true, "rangeBody": true,
	"print": true, "printf": true, "println": true, "html": true, "js": true, "urlquery": true,
}

// gen counts helper output against MaxMockGenerated.
func (s *renderState) gen(v string) (string, error) {
	if s.late() {
		return "", errMockSlow
	}
	s.generated += len(v)
	if s.generated > MaxMockGenerated {
		return "", fmt.Errorf("helpers made more than %d bytes in one response", MaxMockGenerated)
	}
	return v, nil
}

// loop charges n loop items of a body of w steps.
func (s *renderState) loop(w, n int) error {
	if err := s.charge(n); err != nil {
		return err
	}
	s.work += w * n
	if s.work > MaxMockWork {
		return fmt.Errorf("template does more than %d steps in one response (fewer or smaller loops)", MaxMockWork)
	}
	return nil
}

func (s *renderState) funcs() template.FuncMap {
	return template.FuncMap{
		"path":  func(name string) string { return s.req.Params[name] },
		"query": func(name string) string { return s.req.Query.Get(name) },
		"queryAll": func(name string) []string {
			if v := s.req.Query[name]; v != nil {
				return v
			}
			return []string{}
		},
		"header": func(name string) string { return s.req.Header.Get(name) },
		"body":   s.body,
		// Range twins (see rangeFuncs): w is the loop body's step count,
		// decls the number of range variables.
		"rangeRepeat": func(w, decls, n int) (any, error) {
			if err := s.loop(w, max(n, 0)); err != nil {
				return nil, err
			}
			list, err := s.repeat(n)
			if err != nil {
				return nil, err
			}
			k, v := indexed(list)
			return s.items(decls, k, v), nil
		},
		"rangeQueryAll": func(w, decls int, name string) (any, error) {
			list := s.req.Query[name]
			if err := s.loop(w, len(list)); err != nil {
				return nil, err
			}
			k, v := indexed(list)
			return s.items(decls, k, v), nil
		},
		"rangeBody": func(w, decls int, path string) (any, error) {
			switch x := s.body(path).(type) {
			case []any:
				if err := s.loop(w, len(x)); err != nil {
					return nil, err
				}
				k, v := indexed(x)
				return s.items(decls, k, v), nil
			case map[string]any:
				if err := s.loop(w, len(x)); err != nil {
					return nil, err
				}
				names := make([]string, 0, len(x))
				for name := range x {
					names = append(names, name)
				}
				sort.Strings(names) // as range over a map
				k, v := make([]any, len(names)), make([]any, len(names))
				for i, name := range names {
					k[i], v[i] = name, x[name]
				}
				return s.items(decls, k, v), nil
			case nil:
				return s.items(decls, nil, nil), nil
			default:
				return x, nil // range reports it cannot iterate
			}
		},
		"bodyRaw": func() string {
			if s.rawBody == nil {
				raw := string(s.req.Body)
				s.rawBody = &raw
			}
			return *s.rawBody
		},
		"method":      func() string { return s.req.Method },
		"requestPath": func() string { return s.req.Path },
		"fake":        s.fake,
		"pick": func(opts ...any) (any, error) {
			if len(opts) == 0 {
				return nil, errors.New("pick needs at least one value")
			}
			return opts[s.faker.IntN(len(opts))], nil
		},
		"repeat": func(n int) ([]int, error) {
			if err := s.charge(max(n, 0)); err != nil {
				return nil, err
			}
			return s.repeat(n)
		},
		"add": func(a, b int) int { return a + b },
		"int": mockInt,
		"now": mockNow,
		"json": func(v any) (string, error) {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetEscapeHTML(false)
			if err := enc.Encode(v); err != nil {
				return "", err
			}
			return s.gen(strings.TrimSuffix(buf.String(), "\n"))
		},
		"default": func(fallback, v any) any {
			if v == nil || v == "" {
				return fallback
			}
			return v
		},
		"upper": func(v string) (string, error) { return s.gen(strings.ToUpper(v)) },
		"lower": func(v string) (string, error) { return s.gen(strings.ToLower(v)) },

		"print":    func(a ...any) (string, error) { return s.gen(fmt.Sprint(a...)) },
		"printf":   func(f string, a ...any) (string, error) { return s.gen(fmt.Sprintf(f, a...)) },
		"println":  func(a ...any) (string, error) { return s.gen(fmt.Sprintln(a...)) },
		"html":     func(a ...any) (string, error) { return s.gen(template.HTMLEscaper(a...)) },
		"js":       func(a ...any) (string, error) { return s.gen(template.JSEscaper(a...)) },
		"urlquery": func(a ...any) (string, error) { return s.gen(template.URLQueryEscaper(a...)) },
	}
}

func mockInt(v any) (int, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case int:
		return x, nil
	case int64:
		return int(x), nil
	case float64:
		return int(x), nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		x = strings.TrimSpace(x)
		if x == "" {
			return 0, nil
		}
		if n, err := strconv.Atoi(x); err == nil {
			return n, nil
		}
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return int(f), nil
		}
		return 0, fmt.Errorf("int: %q is not a number", x)
	}
	return 0, fmt.Errorf("int: cannot convert %T", v)
}

func mockNow(layout ...string) (any, error) {
	now := time.Now().UTC()
	if len(layout) == 0 {
		return now.Format(time.RFC3339), nil
	}
	if len(layout) > 1 {
		return nil, errors.New("now takes at most one layout")
	}
	if layout[0] == "unix" {
		return now.Unix(), nil
	}
	return now.Format(layout[0]), nil
}

func (s *renderState) repeat(n int) ([]int, error) {
	if n < 0 {
		return nil, errors.New("repeat count must not be negative")
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out, nil
}

// charge counts n loop items against MaxMockRepeat.
func (s *renderState) charge(n int) error {
	s.repeats += n
	if s.repeats > MaxMockRepeat {
		return fmt.Errorf("repeat: more than %d loop items in one response", MaxMockRepeat)
	}
	return nil
}

// body returns the JSON body value at a dot path; nil when missing.
func (s *renderState) body(path string) any {
	if !s.req.parsed {
		s.req.parsed = true
		if len(s.req.Body) > 0 {
			_ = json.Unmarshal(s.req.Body, &s.req.bodyVal)
		}
	}
	v := s.req.bodyVal
	if path == "" {
		return v
	}
	for _, key := range strings.Split(path, ".") {
		switch node := v.(type) {
		case map[string]any:
			v = node[key]
		case []any:
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(node) {
				return nil
			}
			v = node[i]
		default:
			return nil
		}
	}
	return v
}

// fake runs a gofakeit generator; args fill its parameters in order, and the
// last parameter takes the rest when it is a list.
func (s *renderState) fake(name string, args ...any) (any, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	info := gofakeit.GetFuncLookup(name)
	if info == nil || !fakeAllowed(name, info) {
		return nil, fmt.Errorf("fake: unknown generator %q (list: GET /api/mocks/helpers)", name)
	}
	if len(args) > len(info.Params) && (len(info.Params) == 0 || !strings.HasPrefix(info.Params[len(info.Params)-1].Type, "[]")) {
		return nil, fmt.Errorf("fake %q takes at most %d arguments", name, len(info.Params))
	}
	var params *gofakeit.MapParams
	if len(args) > 0 {
		params = gofakeit.NewMapParams()
		for i, a := range args {
			p := info.Params[min(i, len(info.Params)-1)]
			v := fmt.Sprint(a)
			if isCountParam(p.Field) {
				if n, err := strconv.Atoi(v); err == nil && n > maxFakeCount {
					return nil, fmt.Errorf("fake %q: %s must be at most %d", name, p.Field, maxFakeCount)
				}
			}
			params.Add(p.Field, v)
		}
		// gofakeit fills a missing parameter from its default only without a
		// params map, and never splits a list default ("[a,b]"): do both here.
		for _, p := range info.Params[min(len(args), len(info.Params)):] {
			if strings.HasPrefix(p.Default, "[") && strings.HasSuffix(p.Default, "]") {
				for _, d := range strings.Split(p.Default[1:len(p.Default)-1], ",") {
					params.Add(p.Field, d)
				}
			}
		}
	}
	v, err := info.Generate(s.faker, params, info)
	if err != nil {
		return nil, err
	}
	if str, ok := v.(string); ok {
		_, err = s.gen(str)
	} else {
		_, err = s.gen(fmt.Sprint(v))
	}
	return v, err
}

func isCountParam(field string) bool {
	f := strings.ToLower(field)
	return strings.Contains(f, "count") || f == "length" || f == "words" || f == "numdice"
}

// limitedBuffer fails a write past max, which stops the template.
type limitedBuffer struct {
	bytes.Buffer
	max int
}

var errMockTooLarge = fmt.Errorf("response is larger than %d bytes", MaxMockBody)

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, errMockTooLarge
	}
	return b.Buffer.Write(p)
}

// rangeFuncs are the helpers a range may iterate over, each mapped to an
// internal twin that charges the list's length to the repeat budget.
var rangeFuncs = map[string]string{"repeat": "rangeRepeat", "queryAll": "rangeQueryAll", "body": "rangeBody"}

// parseMockTemplate parses a template source and checks that its run is
// bounded: no define/template/block (recursion), and range only over repeat,
// queryAll or body, whose lengths count against MaxMockRepeat. funcs are the
// helpers to bind; nil binds a state with no request (validation only).
func parseMockTemplate(name, src string, funcs template.FuncMap) (*template.Template, error) {
	if funcs == nil {
		funcs = (&renderState{req: &mockRequest{}}).funcs()
	}
	t, err := template.New(name).Option("missingkey=zero").Funcs(funcs).Parse(src)
	if err != nil {
		return nil, err
	}
	if len(t.Templates()) > 1 {
		return nil, errors.New("define and block are not supported")
	}
	if t.Tree == nil {
		return t, nil
	}
	if err := checkMockNode(t.Tree.Root); err != nil {
		return nil, err
	}
	return t, nil
}

func checkMockNode(n parse.Node) error {
	switch n := n.(type) {
	case *parse.ListNode:
		if n == nil {
			return nil
		}
		for _, c := range n.Nodes {
			if err := checkMockNode(c); err != nil {
				return err
			}
		}
	case *parse.TemplateNode:
		return errors.New("template calls are not supported")
	case *parse.IfNode:
		return checkMockBranch(&n.BranchNode)
	case *parse.WithNode:
		return checkMockBranch(&n.BranchNode)
	case *parse.RangeNode:
		cmds := n.Pipe.Cmds
		var ident *parse.IdentifierNode
		if len(cmds) == 1 && len(cmds[0].Args) > 0 {
			ident, _ = cmds[0].Args[0].(*parse.IdentifierNode)
		}
		if ident == nil || rangeFuncs[ident.Ident] == "" {
			return errors.New(`range works only over repeat, queryAll or body, e.g. {{ range $i := repeat 3 }}`)
		}
		// Rewrite `range f args` to `range rangeF w decls args`: each item
		// costs w steps.
		w := max(1, mockSteps(n.List)+mockSteps(n.ElseList))
		ident.Ident = rangeFuncs[ident.Ident]
		num := func(v int) parse.Node {
			return &parse.NumberNode{NodeType: parse.NodeNumber, Pos: ident.Pos, IsInt: true, Int64: int64(v), Text: strconv.Itoa(v)}
		}
		args := append([]parse.Node{ident, num(w), num(len(n.Pipe.Decl))}, cmds[0].Args[1:]...)
		cmds[0].Args = args
		return checkMockBranch(&n.BranchNode)
	}
	return nil
}

// mockSteps counts the steps of a list: one per node and per command; a
// nested range counts its own head only (its items are charged when it runs).
func mockSteps(l *parse.ListNode) int {
	if l == nil {
		return 0
	}
	n := 0
	for _, node := range l.Nodes {
		n++
		switch x := node.(type) {
		case *parse.ActionNode:
			n += len(x.Pipe.Cmds)
		case *parse.IfNode:
			n += len(x.Pipe.Cmds) + mockSteps(x.List) + mockSteps(x.ElseList)
		case *parse.WithNode:
			n += len(x.Pipe.Cmds) + mockSteps(x.List) + mockSteps(x.ElseList)
		case *parse.RangeNode:
			n += len(x.Pipe.Cmds)
		}
	}
	return n
}

func checkMockBranch(b *parse.BranchNode) error {
	if err := checkMockNode(b.List); err != nil {
		return err
	}
	if b.ElseList != nil {
		return checkMockNode(b.ElseList)
	}
	return nil
}

// mockRendered is a rule's response after its templates ran.
type mockRendered struct {
	Headers map[string]string
	Body    []byte
}

// renderMockRule runs the body and header templates of a rule for a request.
// With a seed the fake data is the same on every request.
func renderMockRule(rule *MockRule, req *mockRequest) (mockRendered, error) {
	st := &renderState{req: req, faker: gofakeit.New(uint64(rule.Seed)), deadline: time.Now().Add(MaxMockRenderTime)}
	funcs := st.funcs()
	exec := func(name, src string) ([]byte, error) {
		t, err := parseMockTemplate(name, src, funcs)
		if err != nil {
			return nil, err
		}
		buf := &limitedBuffer{max: MaxMockBody}
		if err := t.Execute(buf, nil); err != nil {
			return nil, err
		}
		if st.timedOut {
			return nil, errMockSlow
		}
		return buf.Bytes(), nil
	}

	out := mockRendered{Headers: make(map[string]string, len(rule.Headers))}
	keys := make([]string, 0, len(rule.Headers))
	for k := range rule.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys) // the same seed gives the same headers
	for _, k := range keys {
		v, err := exec("header "+k, rule.Headers[k])
		if err != nil {
			return mockRendered{}, err
		}
		if strings.ContainsAny(string(v), "\r\n") {
			return mockRendered{}, fmt.Errorf("header %s: value has a line break", k)
		}
		out.Headers[k] = string(v)
	}
	body, err := exec("body", rule.Body)
	if err != nil {
		return mockRendered{}, err
	}
	out.Body = body
	return out, nil
}
