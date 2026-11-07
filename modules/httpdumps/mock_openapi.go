package httpdumps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// OpenAPI import (#4493): a spec (3.0 or 3.1, YAML or JSON) becomes one rule
// per operation. The response is built once, at import: the spec's example
// as a fixed body, else a template from the schema with fake data. The rules
// are ordinary rules of one group; a re-import replaces the group.

const (
	// MaxOpenAPISpec caps a spec, uploaded or downloaded.
	MaxOpenAPISpec = 5 << 20
	// openAPIFetchTimeout bounds downloading a spec by URL.
	openAPIFetchTimeout = 10 * time.Second
	// maxSchemaDepth stops nested and recursive schemas ($ref cycles).
	maxSchemaDepth = 8
	// maxGenItems caps the items generated for an array.
	maxGenItems = 20
)

// OpenAPIImport is what to import and how.
type OpenAPIImport struct {
	Spec    string `json:"spec"`    // the spec text (YAML or JSON)
	URL     string `json:"url"`     // or where to download it (http/https)
	Project string `json:"project"` // project of the rules; empty = every project
	Group   string `json:"group"`   // group name; default "openapi:<title>"
	// BasePath goes before every path; nil = the path of servers[0].url
	// ("/v1" from "https://api.example.com/v1"), "" = none.
	BasePath *string `json:"base_path"`
}

// OpenAPISkip is an operation the import could not turn into a rule.
type OpenAPISkip struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// OpenAPIResult is the outcome of an import.
type OpenAPIResult struct {
	Group   string        `json:"group"`
	Title   string        `json:"title"`
	Rules   []MockRule    `json:"rules"`
	Skipped []OpenAPISkip `json:"skipped"`
}

var errOpenAPI = errors.New("openapi")

// fetchOpenAPI downloads a spec: http/https only, at most MaxOpenAPISpec,
// within openAPIFetchTimeout.
func fetchOpenAPI(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%w: url must be http or https", errOpenAPI)
	}
	ctx, cancel := context.WithTimeout(ctx, openAPIFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errOpenAPI, err)
	}
	req.Header.Set("Accept", "application/yaml, application/json;q=0.9, */*;q=0.5")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: download: %v", errOpenAPI, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: download: %s answered %d", errOpenAPI, u.Host, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxOpenAPISpec+1))
	if err != nil {
		return nil, fmt.Errorf("%w: download: %v", errOpenAPI, err)
	}
	if len(data) > MaxOpenAPISpec {
		return nil, fmt.Errorf("%w: spec is larger than 5 MB", errOpenAPI)
	}
	return data, nil
}

// omap is a YAML/JSON object that keeps its key order, so generated bodies
// list fields as the spec does.
type omap struct {
	keys []string
	m    map[string]any
}

func (o *omap) get(k string) any {
	if o == nil {
		return nil
	}
	return o.m[k]
}

func (o *omap) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteByte(':')
		vb, err := marshalJSON(o.m[k], "")
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// decodeYAMLNode turns a YAML node into Go values with *omap for mappings.
func decodeYAMLNode(n *yaml.Node, depth int) (any, error) {
	if depth > 200 {
		return nil, errors.New("document nests too deep")
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return decodeYAMLNode(n.Content[0], depth+1)
	case yaml.MappingNode:
		o := &omap{m: make(map[string]any, len(n.Content)/2)}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k := n.Content[i].Value
			v, err := decodeYAMLNode(n.Content[i+1], depth+1)
			if err != nil {
				return nil, err
			}
			if _, dup := o.m[k]; !dup {
				o.keys = append(o.keys, k)
			}
			o.m[k] = v
		}
		return o, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := decodeYAMLNode(c, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.AliasNode:
		return nil, errors.New("YAML aliases are not supported")
	default:
		var v any
		if err := n.Decode(&v); err != nil {
			return nil, err
		}
		return v, nil
	}
}

// marshalJSON writes v as JSON without HTML escaping, indented when indent != "".
func marshalJSON(v any, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

var openAPIMethods = []string{"get", "post", "put", "patch", "delete", "head", "options"}

// buildOpenAPIRules turns a spec into rules (not stored, normalized).
func buildOpenAPIRules(spec []byte, in OpenAPIImport) (OpenAPIResult, error) {
	var node yaml.Node
	if err := yaml.Unmarshal(spec, &node); err != nil {
		return OpenAPIResult{}, fmt.Errorf("%w: not YAML or JSON: %v", errOpenAPI, err)
	}
	v, err := decodeYAMLNode(&node, 0)
	if err != nil {
		return OpenAPIResult{}, fmt.Errorf("%w: %v", errOpenAPI, err)
	}
	root, _ := v.(*omap)
	if root == nil {
		return OpenAPIResult{}, fmt.Errorf("%w: the spec is not an object", errOpenAPI)
	}
	if root.get("swagger") != nil {
		return OpenAPIResult{}, fmt.Errorf("%w: Swagger 2.0 is not supported; convert it to OpenAPI 3", errOpenAPI)
	}
	if ver := fmt.Sprint(root.get("openapi")); !strings.HasPrefix(ver, "3.") {
		return OpenAPIResult{}, fmt.Errorf("%w: not an OpenAPI 3 document (no openapi: 3.x)", errOpenAPI)
	}
	paths, _ := root.get("paths").(*omap)
	if paths == nil || len(paths.keys) == 0 {
		return OpenAPIResult{}, fmt.Errorf("%w: the spec has no paths", errOpenAPI)
	}

	g := &schemaGen{root: root}
	info, _ := root.get("info").(*omap)
	res := OpenAPIResult{Title: strings.TrimSpace(fmt.Sprint(info.get("title"))), Rules: []MockRule{}, Skipped: []OpenAPISkip{}}
	if info.get("title") == nil || res.Title == "" {
		res.Title = "spec"
	}
	res.Group = strings.TrimSpace(in.Group)
	if res.Group == "" {
		res.Group = "openapi:" + res.Title
	}
	base := openAPIBasePath(root, in.BasePath)

	for _, p := range paths.keys {
		item, _ := g.resolve(paths.get(p)).(*omap)
		for _, method := range openAPIMethods {
			op, _ := item.get(method).(*omap)
			if op == nil {
				continue
			}
			rule, err := g.rule(base+p, strings.ToUpper(method), op, in.Project)
			if err == nil {
				err = rule.Normalize()
			}
			if err != nil {
				res.Skipped = append(res.Skipped, OpenAPISkip{Method: strings.ToUpper(method), Path: base + p, Reason: err.Error()})
				continue
			}
			res.Rules = append(res.Rules, rule)
		}
	}
	return res, nil
}

func openAPIBasePath(root *omap, override *string) string {
	base := ""
	if override != nil {
		base = *override
	} else if servers, _ := root.get("servers").([]any); len(servers) > 0 {
		if s, _ := servers[0].(*omap); s != nil {
			raw, _ := s.get("url").(string)
			if u, err := url.Parse(raw); err == nil && !strings.Contains(raw, "{") {
				base = u.Path
			}
		}
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base != "" && !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	return base
}

// schemaGen builds rules and body templates from one spec.
type schemaGen struct {
	root       *omap
	pathParams []string
}

// resolve follows local $refs ("#/components/schemas/User"); an external or
// broken ref resolves to nil.
func (g *schemaGen) resolve(v any) any {
	for hops := 0; hops < 32; hops++ {
		o, ok := v.(*omap)
		if !ok {
			return v
		}
		ref, ok := o.get("$ref").(string)
		if !ok {
			return v
		}
		if !strings.HasPrefix(ref, "#/") {
			return nil
		}
		var cur any = g.root
		for _, part := range strings.Split(ref[2:], "/") {
			part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			if p, err := url.PathUnescape(part); err == nil {
				part = p
			}
			co, _ := cur.(*omap)
			if co == nil {
				return nil
			}
			cur = co.get(part)
		}
		v = cur
	}
	return nil
}

func (g *schemaGen) rule(path, method string, op *omap, project string) (MockRule, error) {
	rule := MockRule{Method: method, Path: path, Project: project, Enabled: true, Headers: map[string]string{}}
	rule.Name = method + " " + path
	if id, _ := op.get("operationId").(string); id != "" {
		rule.Name = id
	} else if s, _ := op.get("summary").(string); s != "" {
		rule.Name = s
	}
	g.pathParams = nil
	for _, seg := range splitPath(path) {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			g.pathParams = append(g.pathParams, seg[1:len(seg)-1])
		}
	}

	responses, _ := op.get("responses").(*omap)
	if responses == nil || len(responses.keys) == 0 {
		return rule, errors.New("no responses")
	}
	code, resp := pickResponse(responses)
	rule.Status = code
	r, _ := g.resolve(resp).(*omap)
	content, _ := r.get("content").(*omap)
	if content == nil || len(content.keys) == 0 || code == http.StatusNoContent || method == http.MethodHead {
		return rule, nil
	}
	mediaType := content.keys[0]
	for _, k := range content.keys {
		if strings.Contains(strings.ToLower(k), "json") {
			mediaType = k
			break
		}
	}
	media, _ := g.resolve(content.get(mediaType)).(*omap)
	isJSON := strings.Contains(strings.ToLower(mediaType), "json")
	if !strings.Contains(mediaType, "*") {
		rule.Headers["Content-Type"] = mediaType
	}

	if ex, ok := g.example(media); ok {
		body, err := exampleBody(ex, isJSON)
		if err != nil {
			return rule, err
		}
		rule.Body = body
		return rule, nil
	}
	schema := g.resolve(media.get("schema"))
	if schema == nil {
		return rule, nil
	}
	if so, _ := schema.(*omap); so != nil {
		if ex, ok := so.m["example"]; ok {
			body, err := exampleBody(ex, isJSON)
			rule.Body = body
			return rule, err
		}
	}
	if isJSON {
		rule.Body = g.value(schema, "", "", false, 0)
		rule.Template = true
	} else if typeOf(schema) == "string" {
		rule.Body = `{{ fake "sentence" }}`
		rule.Template = true
	}
	return rule, nil
}

// pickResponse takes the first success: the lowest 2xx, then 2XX, then
// default (as 200), then the lowest code there is.
func pickResponse(responses *omap) (int, any) {
	best, bestCode := any(nil), 0
	for _, k := range responses.keys {
		if c, err := strconv.Atoi(k); err == nil && c >= 200 && c < 300 && (bestCode == 0 || c < bestCode) {
			best, bestCode = responses.get(k), c
		}
	}
	if best != nil {
		return bestCode, best
	}
	for _, k := range []string{"2XX", "2xx", "default"} {
		if v := responses.get(k); v != nil {
			return http.StatusOK, v
		}
	}
	for _, k := range responses.keys {
		if c, err := strconv.Atoi(k); err == nil && c >= 100 && c <= 599 && (bestCode == 0 || c < bestCode) {
			best, bestCode = responses.get(k), c
		}
	}
	if best != nil {
		return bestCode, best
	}
	return http.StatusOK, responses.get(responses.keys[0])
}

// example returns a media type's example: example, else the first of examples.
func (g *schemaGen) example(media *omap) (any, bool) {
	if media == nil {
		return nil, false
	}
	if ex, ok := media.m["example"]; ok {
		return ex, true
	}
	if exs, _ := media.get("examples").(*omap); exs != nil && len(exs.keys) > 0 {
		if e, _ := g.resolve(exs.get(exs.keys[0])).(*omap); e != nil {
			if v, ok := e.m["value"]; ok {
				return v, true
			}
		}
	}
	return nil, false
}

func exampleBody(ex any, isJSON bool) (string, error) {
	if s, ok := ex.(string); ok && !isJSON {
		return s, nil
	}
	b, err := marshalJSON(ex, "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// typeOf is a schema's type: "type" (the first non-null of a 3.1 list), else
// guessed from properties or items.
func typeOf(v any) string {
	s, _ := v.(*omap)
	if s == nil {
		return ""
	}
	switch t := s.get("type").(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if xs, _ := x.(string); xs != "" && xs != "null" {
				return xs
			}
		}
		return "null"
	}
	if s.get("properties") != nil {
		return "object"
	}
	if s.get("items") != nil {
		return "array"
	}
	return ""
}

// fakeByName picks a generator from a string property's name.
var fakeByName = map[string]string{
	"email": "email", "mail": "email", "emailaddress": "email",
	"firstname": "firstname", "givenname": "firstname",
	"lastname": "lastname", "surname": "lastname", "familyname": "lastname",
	"name": "name", "fullname": "name", "displayname": "name",
	"username": "username", "login": "username", "nickname": "username",
	"phone": "phone", "phonenumber": "phone", "mobile": "phone", "telephone": "phone",
	"city": "city", "country": "country", "countrycode": "countryabr", "state": "state",
	"street": "street", "address": "street", "addressline": "street", "address1": "street",
	"zip": "zip", "zipcode": "zip", "postcode": "zip", "postalcode": "zip",
	"company": "company", "companyname": "company", "organization": "company",
	"url": "url", "website": "url", "homepage": "url", "link": "url", "avatar": "url", "image": "url",
	"title": "sentence", "subject": "sentence",
	"description": "sentence", "summary": "sentence", "comment": "sentence", "message": "sentence", "text": "sentence", "note": "sentence",
	"currency": "currencyshort", "color": "color", "colour": "color", "language": "language",
	"jobtitle": "jobtitle", "ip": "ipv4address", "ipaddress": "ipv4address", "useragent": "useragent",
}

var fakeByFormat = map[string]string{
	"email": "email", "idn-email": "email", "uuid": "uuid",
	"uri": "url", "url": "url", "uri-reference": "url", "iri": "url",
	"hostname": "domainname", "idn-hostname": "domainname",
	"ipv4": "ipv4address", "ipv6": "ipv6address",
}

// value is the template of one JSON value of schema. indent is the current
// line indent; prop is the field name (for fake data by name); rootField
// marks a field of the response's root object, which path params fill.
func (g *schemaGen) value(schema any, indent, prop string, rootField bool, depth int) string {
	s, _ := g.resolve(schema).(*omap)
	if s == nil || depth > maxSchemaDepth {
		return "null"
	}
	if all, _ := s.get("allOf").([]any); len(all) > 0 {
		s = g.mergeAllOf(s, all, depth)
	}
	for _, k := range []string{"oneOf", "anyOf"} {
		if alts, _ := s.get(k).([]any); len(alts) > 0 {
			return g.value(alts[0], indent, prop, rootField, depth+1)
		}
	}
	// A root field named like a path parameter echoes it, over any example.
	if param := g.pathParam(prop); rootField && param != "" {
		switch typeOf(s) {
		case "string":
			return fmt.Sprintf(`{{ path %q | json }}`, param)
		case "integer", "number":
			return fmt.Sprintf(`{{ int (path %q) }}`, param)
		}
	}
	if ex, ok := s.m["example"]; ok {
		return jsonLiteral(ex, indent)
	}
	if exs, _ := s.get("examples").([]any); len(exs) > 0 {
		return jsonLiteral(exs[0], indent)
	}
	if c, ok := s.m["const"]; ok {
		return jsonLiteral(c, indent)
	}
	if enum, _ := s.get("enum").([]any); len(enum) > 0 {
		return enumTemplate(enum, indent)
	}

	t := typeOf(s)
	switch t {
	case "object":
		props, _ := s.get("properties").(*omap)
		if props == nil || len(props.keys) == 0 {
			return "{}"
		}
		inner := indent + "  "
		var b strings.Builder
		b.WriteString("{\n")
		for i, k := range props.keys {
			key, _ := marshalJSON(k, "")
			b.WriteString(inner + string(key) + ": ")
			b.WriteString(g.value(props.get(k), inner, k, depth == 0, depth+1))
			if i < len(props.keys)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(indent + "}")
		return b.String()
	case "array":
		n := 2
		if minI := intOf(s.get("minItems")); minI > 0 {
			n = minI
		}
		if maxI := intOf(s.get("maxItems")); maxI > 0 && n > maxI {
			n = maxI
		}
		n = min(n, maxGenItems)
		if n == 0 {
			return "[]"
		}
		inner := indent + "  "
		return fmt.Sprintf("[{{ range $i := repeat %d }}{{ if $i }},{{ end }}\n%s%s{{ end }}\n%s]",
			n, inner, g.value(s.get("items"), inner, "", false, depth+1), indent)
	case "string":
		format, _ := s.get("format").(string)
		switch format {
		case "date-time":
			return `{{ fake "date" | json }}`
		case "date":
			return `{{ now "2006-01-02" | json }}`
		case "time":
			return `{{ now "15:04:05" | json }}`
		}
		if f := fakeByFormat[format]; f != "" {
			return fmt.Sprintf(`{{ fake %q | json }}`, f)
		}
		if f := fakeByName[normName(prop)]; f != "" {
			return fmt.Sprintf(`{{ fake %q | json }}`, f)
		}
		return `{{ fake "word" | json }}`
	case "integer":
		lo, hi := 1, 1000
		if v, ok := s.m["minimum"]; ok {
			lo = intOf(v)
		}
		if v, ok := s.m["maximum"]; ok {
			hi = intOf(v)
		}
		if hi < lo {
			hi = lo + 1000
		}
		return fmt.Sprintf(`{{ fake "number" %d %d }}`, lo, hi)
	case "number":
		lo, hi := 0.0, 1000.0
		if v, ok := s.m["minimum"]; ok {
			lo = floatOf(v)
		}
		if v, ok := s.m["maximum"]; ok {
			hi = floatOf(v)
		}
		if hi < lo {
			hi = lo + 1000
		}
		return fmt.Sprintf(`{{ fake "float64range" %s %s }}`, strconv.FormatFloat(lo, 'f', -1, 64), strconv.FormatFloat(hi, 'f', -1, 64))
	case "boolean":
		return `{{ fake "bool" }}`
	}
	return "null"
}

// pathParam is the path parameter a top-level field echoes: the same name,
// or "id" for the last parameter.
func (g *schemaGen) pathParam(prop string) string {
	if prop == "" || len(g.pathParams) == 0 {
		return ""
	}
	for _, p := range g.pathParams {
		if p == prop {
			return p
		}
	}
	if prop == "id" {
		return g.pathParams[len(g.pathParams)-1]
	}
	return ""
}

func (g *schemaGen) mergeAllOf(s *omap, all []any, depth int) *omap {
	out := &omap{m: map[string]any{}}
	props := &omap{m: map[string]any{}}
	addProps := func(o *omap) {
		if p, _ := o.get("properties").(*omap); p != nil {
			for _, k := range p.keys {
				if _, ok := props.m[k]; !ok {
					props.keys = append(props.keys, k)
				}
				props.m[k] = p.m[k]
			}
		}
	}
	for _, part := range all {
		ps, _ := g.resolve(part).(*omap)
		if ps == nil {
			continue
		}
		if nested, _ := ps.get("allOf").([]any); len(nested) > 0 && depth < maxSchemaDepth {
			ps = g.mergeAllOf(ps, nested, depth+1)
		}
		addProps(ps)
	}
	addProps(s)
	for _, k := range s.keys {
		if k != "allOf" && k != "properties" {
			out.keys = append(out.keys, k)
			out.m[k] = s.m[k]
		}
	}
	out.keys = append(out.keys, "type", "properties")
	out.m["type"], out.m["properties"] = "object", props
	return out
}

// jsonLiteral writes v as JSON in a template: "{{" in it would start an
// action, so such text goes through a Go string literal and | json.
func jsonLiteral(v any, indent string) string {
	b, err := marshalJSON(v, "  ")
	if err != nil {
		return "null"
	}
	out := strings.ReplaceAll(string(b), "\n", "\n"+indent)
	if strings.Contains(out, "{{") || strings.Contains(out, "}}") {
		compact, _ := marshalJSON(v, "")
		return fmt.Sprintf(`{{ print %s }}`, strconv.Quote(string(compact)))
	}
	return out
}

// enumTemplate picks one of the enum's values at random: plain values as
// template literals through | json, anything else as its JSON text.
func enumTemplate(enum []any, indent string) string {
	if len(enum) == 1 {
		return jsonLiteral(enum[0], indent)
	}
	args := make([]string, len(enum))
	plain := true
	for i, e := range enum {
		switch x := e.(type) {
		case string:
			args[i] = strconv.Quote(x)
		case int, int64, uint64, float64, bool:
			args[i] = fmt.Sprint(x)
		default:
			plain = false
		}
	}
	if plain {
		return fmt.Sprintf(`{{ pick %s | json }}`, strings.Join(args, " "))
	}
	for i, e := range enum {
		b, _ := marshalJSON(e, "")
		args[i] = strconv.Quote(string(b))
	}
	return fmt.Sprintf(`{{ pick %s }}`, strings.Join(args, " "))
}

func normName(s string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "", " ", "").Replace(s))
}

func intOf(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case uint64:
		return int(x)
	case float64:
		return int(x)
	}
	return 0
}

func floatOf(v any) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case uint64:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

// OpenAPIRulesFromFile builds the rules of a spec file (for config rules:
// not stored, read-only once set with SetMockRules).
func OpenAPIRulesFromFile(path string, in OpenAPIImport) (OpenAPIResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OpenAPIResult{}, err
	}
	if len(data) > MaxOpenAPISpec {
		return OpenAPIResult{}, errors.New("spec is larger than 5 MB")
	}
	res, err := buildOpenAPIRules(data, in)
	if err != nil {
		return OpenAPIResult{}, err
	}
	for i := range res.Rules {
		res.Rules[i].Group = res.Group
	}
	return res, nil
}

// importOpenAPI reads the spec (text or URL), builds the rules and replaces
// the group with them.
func importOpenAPI(ctx context.Context, rules *mockStore, in OpenAPIImport, source string) (OpenAPIResult, error) {
	spec := []byte(in.Spec)
	switch {
	case strings.TrimSpace(in.Spec) != "" && in.URL != "":
		return OpenAPIResult{}, ValidationError{"spec": "give spec or url, not both"}
	case strings.TrimSpace(in.Spec) == "" && in.URL == "":
		return OpenAPIResult{}, ValidationError{"spec": "give the spec text or its url"}
	case in.URL != "":
		data, err := fetchOpenAPI(ctx, in.URL)
		if err != nil {
			return OpenAPIResult{}, ValidationError{"url": strings.TrimPrefix(err.Error(), "openapi: ")}
		}
		spec = data
	case len(spec) > MaxOpenAPISpec:
		return OpenAPIResult{}, ValidationError{"spec": "must be at most 5 MB"}
	}
	in.Project = strings.TrimSpace(in.Project)
	res, err := buildOpenAPIRules(spec, in)
	if err != nil {
		return OpenAPIResult{}, ValidationError{"spec": strings.TrimPrefix(err.Error(), "openapi: ")}
	}
	if len(res.Rules) == 0 {
		return res, ValidationError{"spec": "no operation could be imported"}
	}
	stored, err := rules.replaceGroup(res.Group, source, res.Rules)
	if err != nil {
		return OpenAPIResult{}, err
	}
	res.Rules = stored
	return res, nil
}
