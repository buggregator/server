package codegraph

import (
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// phpExtractor walks a PHP tree with its scope (namespace, use imports,
// enclosing class and function), so every call gets its caller and, where the
// source allows it, the full name of its target.
type phpExtractor struct {
	nodeReader
	out *parsedFile

	ns       string
	nsIdx    int
	classUse map[string]string // lower alias -> FQCN
	funcUse  map[string]string // lower alias -> function FQN
	fileEnd  int
}

type phpScope struct {
	classIdx  int
	classFQN  string
	parentFQN string
	fnIdx     int
}

func (s phpScope) from(nsIdx int) int {
	switch {
	case s.fnIdx >= 0:
		return s.fnIdx
	case s.classIdx >= 0:
		return s.classIdx
	}
	return nsIdx
}

func extractPHP(root *ts.Node, src []byte, lang *ts.Language) *parsedFile {
	x := &phpExtractor{
		nodeReader: nodeReader{src: src, lang: lang},
		out:        &parsedFile{HasError: root.HasError()},
		nsIdx:      -1,
		classUse:   map[string]string{},
		funcUse:    map[string]string{},
		fileEnd:    lineEnd(root),
	}
	x.visitChildren(root, phpScope{classIdx: -1, fnIdx: -1})
	return x.out
}

func (x *phpExtractor) add(s parsedSymbol) int {
	x.out.Symbols = append(x.out.Symbols, s)
	return len(x.out.Symbols) - 1
}

func (x *phpExtractor) edge(e parsedEdge) {
	if e.Confidence == "" {
		e.Confidence = ConfExact
	}
	x.out.Edges = append(x.out.Edges, e)
}

func (x *phpExtractor) visitChildren(n *ts.Node, sc phpScope) {
	for _, c := range namedChildren(n) {
		x.visit(c, sc)
	}
}

func (x *phpExtractor) visit(n *ts.Node, sc phpScope) {
	switch x.typ(n) {
	case "namespace_definition":
		x.namespace(n, sc)
		return
	case "namespace_use_declaration":
		x.useDecl(n, sc)
		return
	case "class_declaration", "interface_declaration", "trait_declaration", "enum_declaration":
		x.classLike(n)
		return
	case "function_definition":
		x.function(n, sc)
		return
	case "method_declaration":
		x.method(n, sc)
		return
	case "use_declaration": // trait use inside a class body
		if sc.classIdx >= 0 {
			for _, c := range namedChildren(n) {
				if t := x.typ(c); t == "name" || t == "qualified_name" {
					x.edge(parsedEdge{From: sc.classIdx, Kind: EdgeUses, ToName: lastSegment(x.text(c)), ToFQN: x.resolveClass(x.text(c), sc), ToKind: KindTrait, Line: lineStart(c)})
				}
			}
		}
		return
	case "function_call_expression":
		x.funcCall(n, sc)
	case "member_call_expression", "nullsafe_member_call_expression":
		x.memberCall(n, sc)
	case "scoped_call_expression":
		x.scopedCall(n, sc)
	case "object_creation_expression":
		x.newExpr(n, sc)
	}
	x.visitChildren(n, sc)
}

func (x *phpExtractor) namespace(n *ts.Node, sc phpScope) {
	name := strings.TrimPrefix(x.text(x.field(n, "name")), `\`)
	body := x.field(n, "body")
	x.ns = name
	x.classUse = map[string]string{}
	x.funcUse = map[string]string{}
	end := x.fileEnd
	if body != nil {
		end = lineEnd(n)
	}
	x.nsIdx = -1
	if name != "" {
		x.nsIdx = x.add(parsedSymbol{Kind: KindNamespace, Name: lastSegment(name), FQN: name, Parent: -1, LineStart: lineStart(n), LineEnd: end})
	}
	if body != nil {
		x.visitChildren(body, sc)
		x.ns, x.nsIdx = "", -1
	}
}

func (x *phpExtractor) useDecl(n *ts.Node, sc phpScope) {
	head := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(x.text(n)), "use"))
	kind := KindClass
	switch {
	case strings.HasPrefix(head, "function ") || strings.HasPrefix(head, "function\t"):
		kind = KindFunction
	case strings.HasPrefix(head, "const ") || strings.HasPrefix(head, "const\t"):
		return
	}
	prefix := ""
	var clauses []*ts.Node
	for _, c := range namedChildren(n) {
		switch x.typ(c) {
		case "namespace_name":
			prefix = strings.Trim(x.text(c), `\`)
		case "namespace_use_clause":
			clauses = append(clauses, c)
		case "namespace_use_group":
			for _, g := range namedChildren(c) {
				if x.typ(g) == "namespace_use_clause" {
					clauses = append(clauses, g)
				}
			}
		}
	}
	for _, c := range clauses {
		alias := x.text(x.field(c, "alias"))
		var target string
		for _, cc := range namedChildren(c) {
			if t := x.typ(cc); (t == "qualified_name" || t == "name") && x.field(c, "alias") != cc {
				target = strings.Trim(x.text(cc), `\`)
				break
			}
		}
		if target == "" {
			continue
		}
		if prefix != "" {
			target = prefix + `\` + target
		}
		if alias == "" {
			alias = lastSegment(target)
		}
		if kind == KindFunction {
			x.funcUse[strings.ToLower(alias)] = target
		} else {
			x.classUse[strings.ToLower(alias)] = target
		}
		x.edge(parsedEdge{From: sc.from(x.nsIdx), Kind: EdgeImports, ToName: lastSegment(target), ToFQN: target, ToKind: kind, Line: lineStart(c)})
	}
}

func (x *phpExtractor) qualify(name string) string {
	if x.ns == "" {
		return name
	}
	return x.ns + `\` + name
}

// resolveClass turns a class name as written into its FQCN.
func (x *phpExtractor) resolveClass(name string, sc phpScope) string {
	name = strings.TrimSpace(name)
	if strings.HasPrefix(name, `\`) {
		return name[1:]
	}
	switch strings.ToLower(name) {
	case "self", "static":
		return sc.classFQN
	case "parent":
		return sc.parentFQN
	}
	first, rest, qualified := strings.Cut(name, `\`)
	if strings.EqualFold(first, "namespace") && qualified {
		return x.qualify(rest)
	}
	if full, ok := x.classUse[strings.ToLower(first)]; ok {
		if qualified {
			return full + `\` + rest
		}
		return full
	}
	return x.qualify(name)
}

func (x *phpExtractor) classLike(n *ts.Node) {
	kind := map[string]string{
		"class_declaration":     KindClass,
		"interface_declaration": KindInterface,
		"trait_declaration":     KindTrait,
		"enum_declaration":      KindEnum,
	}[x.typ(n)]
	name := x.text(x.field(n, "name"))
	if name == "" {
		return
	}
	fqn := x.qualify(name)
	body := x.field(n, "body")
	idx := x.add(parsedSymbol{Kind: kind, Name: name, FQN: fqn, Parent: x.nsIdx, LineStart: lineStart(n), LineEnd: lineEnd(n), Signature: x.signature(n, body)})
	sc := phpScope{classIdx: idx, classFQN: fqn, fnIdx: -1}
	for _, c := range namedChildren(n) {
		switch x.typ(c) {
		case "base_clause":
			rel := EdgeExtends
			for _, b := range namedChildren(c) {
				if t := x.typ(b); t == "name" || t == "qualified_name" {
					target := x.resolveClass(x.text(b), sc)
					if kind == KindClass && sc.parentFQN == "" {
						sc.parentFQN = target
					}
					x.edge(parsedEdge{From: idx, Kind: rel, ToName: lastSegment(x.text(b)), ToFQN: target, ToKind: KindClass, Line: lineStart(b)})
				}
			}
		case "class_interface_clause":
			for _, b := range namedChildren(c) {
				if t := x.typ(b); t == "name" || t == "qualified_name" {
					x.edge(parsedEdge{From: idx, Kind: EdgeImplements, ToName: lastSegment(x.text(b)), ToFQN: x.resolveClass(x.text(b), sc), ToKind: KindInterface, Line: lineStart(b)})
				}
			}
		}
	}
	if body != nil {
		x.visitChildren(body, sc)
	}
}

func (x *phpExtractor) function(n *ts.Node, sc phpScope) {
	name := x.text(x.field(n, "name"))
	if name == "" {
		return
	}
	body := x.field(n, "body")
	idx := x.add(parsedSymbol{Kind: KindFunction, Name: name, FQN: x.qualify(name), Parent: x.nsIdx, LineStart: lineStart(n), LineEnd: lineEnd(n), Signature: x.signature(n, body)})
	sc.fnIdx = idx
	sc.classIdx, sc.classFQN, sc.parentFQN = -1, "", ""
	x.visitChildren(n, sc)
}

func (x *phpExtractor) method(n *ts.Node, sc phpScope) {
	name := x.text(x.field(n, "name"))
	if name == "" || sc.classIdx < 0 {
		x.visitChildren(n, sc)
		return
	}
	body := x.field(n, "body")
	idx := x.add(parsedSymbol{Kind: KindMethod, Name: name, FQN: sc.classFQN + "::" + name, Parent: sc.classIdx, LineStart: lineStart(n), LineEnd: lineEnd(n), Signature: x.signature(n, body)})
	sc.fnIdx = idx
	x.visitChildren(n, sc)
}

func (x *phpExtractor) funcCall(n *ts.Node, sc phpScope) {
	fn := x.field(n, "function")
	t := x.typ(fn)
	if t != "name" && t != "qualified_name" {
		return // $f(), (closure)(), ...
	}
	raw := x.text(fn)
	e := parsedEdge{From: sc.from(x.nsIdx), Kind: EdgeCalls, ToName: lastSegment(raw), ToKind: KindFunction, Line: lineStart(n)}
	switch {
	case strings.HasPrefix(raw, `\`):
		e.ToFQN = raw[1:]
	case strings.Contains(raw, `\`):
		first, rest, _ := strings.Cut(raw, `\`)
		if full, ok := x.classUse[strings.ToLower(first)]; ok {
			e.ToFQN = full + `\` + rest
		} else {
			e.ToFQN = x.qualify(raw)
		}
	default:
		if full, ok := x.funcUse[strings.ToLower(raw)]; ok {
			e.ToFQN = full
		} else {
			// PHP looks in the current namespace first, then globally.
			e.ToFQN = x.qualify(raw)
			if x.ns != "" {
				e.AltFQN = raw
			}
		}
	}
	x.edge(e)
}

func (x *phpExtractor) memberCall(n *ts.Node, sc phpScope) {
	nameNode := x.field(n, "name")
	if x.typ(nameNode) != "name" {
		return // $obj->$method()
	}
	name := x.text(nameNode)
	e := parsedEdge{From: sc.from(x.nsIdx), Kind: EdgeCalls, ToName: name, ToKind: KindMethod, Line: lineStart(n), Confidence: ConfName}
	if obj := x.field(n, "object"); x.typ(obj) == "variable_name" && x.text(obj) == "$this" && sc.classFQN != "" {
		e.ToFQN = sc.classFQN + "::" + name
		e.Confidence = ConfExact
	}
	x.edge(e)
}

func (x *phpExtractor) scopedCall(n *ts.Node, sc phpScope) {
	nameNode := x.field(n, "name")
	if x.typ(nameNode) != "name" {
		return
	}
	name := x.text(nameNode)
	e := parsedEdge{From: sc.from(x.nsIdx), Kind: EdgeCalls, ToName: name, ToKind: KindMethod, Line: lineStart(n), Confidence: ConfName}
	scope := x.field(n, "scope")
	switch x.typ(scope) {
	case "relative_scope", "name", "qualified_name":
		if class := x.resolveClass(x.text(scope), sc); class != "" {
			e.ToFQN = class + "::" + name
			e.Confidence = ConfExact
		}
	}
	x.edge(e)
}

func (x *phpExtractor) newExpr(n *ts.Node, sc phpScope) {
	var cls *ts.Node
	for _, c := range namedChildren(n) {
		if t := x.typ(c); t == "name" || t == "qualified_name" {
			cls = c
			break
		}
		if t := x.typ(c); t != "attribute_list" {
			break // new $class, new class {...}
		}
	}
	if cls == nil {
		return
	}
	target := x.resolveClass(x.text(cls), sc)
	if target == "" {
		return
	}
	x.edge(parsedEdge{From: sc.from(x.nsIdx), Kind: EdgeInstantiates, ToName: lastSegment(target), ToFQN: target, ToKind: KindClass, Line: lineStart(n)})
}

// signature is the declaration's text up to its body, on one line.
func (x *phpExtractor) signature(n, body *ts.Node) string {
	return signatureText(x.src, n, body)
}

func signatureText(src []byte, n, body *ts.Node) string {
	end := n.EndByte()
	if body != nil {
		end = body.StartByte()
	}
	s := string(src[n.StartByte():end])
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(strings.TrimSuffix(s, "{"))
	if len(s) > 300 {
		s = s[:300]
	}
	return s
}

func lastSegment(name string) string {
	name = strings.TrimRight(name, `\`)
	if i := strings.LastIndexAny(name, `\`); i >= 0 {
		return name[i+1:]
	}
	return name
}
