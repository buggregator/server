package codegraph

import (
	"regexp"
	"strconv"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
)

// goExtractor walks a Go tree. pkg is the import path of the file's package;
// FQNs are pkg.Func and pkg.Type.Method.
type goExtractor struct {
	nodeReader
	out     *parsedFile
	pkg     string
	pkgIdx  int
	imports map[string]string // local name -> import path
	types   map[string]int    // type name -> symbol index (this file)
}

type goScope struct {
	fnIdx    int
	recvName string
	recvType string
}

var goBuiltins = map[string]bool{
	"append": true, "cap": true, "clear": true, "close": true, "complex": true, "copy": true, "delete": true,
	"imag": true, "len": true, "make": true, "max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true,
	"bool": true, "byte": true, "complex64": true, "complex128": true, "error": true, "float32": true, "float64": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true, "rune": true, "string": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "uintptr": true, "any": true,
}

func extractGo(root *ts.Node, src []byte, lang *ts.Language, pkg string) *parsedFile {
	x := &goExtractor{
		nodeReader: nodeReader{src: src, lang: lang},
		out:        &parsedFile{HasError: root.HasError()},
		pkg:        pkg,
		pkgIdx:     -1,
		imports:    map[string]string{},
		types:      map[string]int{},
	}
	children := namedChildren(root)
	// Package, imports and types first: methods may come before their type.
	for _, c := range children {
		switch x.typ(c) {
		case "package_clause":
			name := ""
			for _, cc := range namedChildren(c) {
				name = x.text(cc)
			}
			if x.pkg == "" {
				x.pkg = name
			}
			x.pkgIdx = x.add(parsedSymbol{Kind: KindPackage, Name: name, FQN: x.pkg, Parent: -1, LineStart: 1, LineEnd: lineEnd(root)})
		case "import_declaration":
			x.importDecl(c)
		case "type_declaration":
			x.typeDecl(c)
		}
	}
	for _, c := range children {
		switch x.typ(c) {
		case "function_declaration", "method_declaration":
			x.funcDecl(c)
		case "var_declaration", "const_declaration":
			x.visit(c, goScope{fnIdx: -1})
		}
	}
	return x.out
}

func (x *goExtractor) add(s parsedSymbol) int {
	x.out.Symbols = append(x.out.Symbols, s)
	return len(x.out.Symbols) - 1
}

func (x *goExtractor) edge(e parsedEdge) {
	if e.Confidence == "" {
		e.Confidence = ConfExact
	}
	x.out.Edges = append(x.out.Edges, e)
}

func (x *goExtractor) qualify(name string) string { return x.pkg + "." + name }

var goMajorSuffix = regexp.MustCompile(`^v[0-9]+$`)

// goImportName guesses the package name of an import path: the last element,
// without a major-version element (/v2) or suffix (yaml.v3) and go-/-go.
func goImportName(path string) string {
	parts := strings.Split(path, "/")
	name := parts[len(parts)-1]
	if goMajorSuffix.MatchString(name) && len(parts) > 1 {
		name = parts[len(parts)-2]
	}
	if i := strings.Index(name, ".v"); i > 0 && goMajorSuffix.MatchString(name[i+1:]) {
		name = name[:i]
	}
	name = strings.TrimSuffix(strings.TrimPrefix(name, "go-"), "-go")
	return strings.ReplaceAll(name, "-", "_")
}

func (x *goExtractor) importDecl(n *ts.Node) {
	var specs []*ts.Node
	for _, c := range namedChildren(n) {
		switch x.typ(c) {
		case "import_spec":
			specs = append(specs, c)
		case "import_spec_list":
			for _, s := range namedChildren(c) {
				if x.typ(s) == "import_spec" {
					specs = append(specs, s)
				}
			}
		}
	}
	for _, s := range specs {
		path, err := strconv.Unquote(x.text(x.field(s, "path")))
		if err != nil || path == "" {
			continue
		}
		local := x.text(x.field(s, "name"))
		if local == "" {
			local = goImportName(path)
		}
		if local != "_" && local != "." {
			x.imports[local] = path
		}
		x.edge(parsedEdge{From: x.pkgIdx, Kind: EdgeImports, ToName: goImportName(path), ToFQN: path, ToKind: KindPackage, Line: lineStart(s)})
	}
}

func (x *goExtractor) typeDecl(n *ts.Node) {
	for _, spec := range namedChildren(n) {
		if t := x.typ(spec); t != "type_spec" && t != "type_alias" {
			continue
		}
		name := x.text(x.field(spec, "name"))
		if name == "" {
			continue
		}
		kind := KindType
		switch x.typ(x.field(spec, "type")) {
		case "struct_type":
			kind = KindStruct
		case "interface_type":
			kind = KindInterface
		}
		x.types[name] = x.add(parsedSymbol{Kind: kind, Name: name, FQN: x.qualify(name), Parent: x.pkgIdx, LineStart: lineStart(spec), LineEnd: lineEnd(spec), Signature: "type " + signatureText(x.src, spec, nil)})
	}
}

// receiverType returns the receiver variable and its type name: (s *S[T]) -> s, S.
func (x *goExtractor) receiverType(n *ts.Node) (string, string) {
	recv := x.field(n, "receiver")
	for _, p := range namedChildren(recv) {
		if x.typ(p) != "parameter_declaration" {
			continue
		}
		name := x.text(x.field(p, "name"))
		t := x.field(p, "type")
		for t != nil {
			switch x.typ(t) {
			case "pointer_type":
				kids := namedChildren(t)
				if len(kids) == 0 {
					return name, ""
				}
				t = kids[0]
				continue
			case "generic_type":
				t = x.field(t, "type")
				continue
			case "type_identifier":
				return name, x.text(t)
			}
			return name, ""
		}
		return name, ""
	}
	return "", ""
}

func (x *goExtractor) funcDecl(n *ts.Node) {
	name := x.text(x.field(n, "name"))
	if name == "" {
		return
	}
	body := x.field(n, "body")
	s := parsedSymbol{Kind: KindFunction, Name: name, FQN: x.qualify(name), Parent: x.pkgIdx, LineStart: lineStart(n), LineEnd: lineEnd(n), Signature: signatureText(x.src, n, body)}
	sc := goScope{}
	if x.typ(n) == "method_declaration" {
		recvName, recvType := x.receiverType(n)
		if recvType == "" {
			return
		}
		s.Kind = KindMethod
		s.FQN = x.qualify(recvType + "." + name)
		if idx, ok := x.types[recvType]; ok {
			s.Parent = idx
		}
		sc.recvName, sc.recvType = recvName, recvType
	}
	sc.fnIdx = x.add(s)
	x.visit(body, sc)
}

func (x *goExtractor) visit(n *ts.Node, sc goScope) {
	if n == nil {
		return
	}
	switch x.typ(n) {
	case "call_expression":
		x.call(n, sc)
	case "composite_literal":
		x.literal(n, sc)
	}
	for _, c := range namedChildren(n) {
		x.visit(c, sc)
	}
}

func (x *goExtractor) from(sc goScope) int {
	if sc.fnIdx >= 0 {
		return sc.fnIdx
	}
	return x.pkgIdx
}

func (x *goExtractor) call(n *ts.Node, sc goScope) {
	fn := x.field(n, "function")
	switch x.typ(fn) {
	case "generic_type", "index_expression": // F[T](...)
		if inner := namedChildren(fn); len(inner) > 0 {
			fn = inner[0]
		}
	}
	e := parsedEdge{From: x.from(sc), Kind: EdgeCalls, Line: lineStart(n)}
	switch x.typ(fn) {
	case "identifier":
		name := x.text(fn)
		if goBuiltins[name] {
			return
		}
		e.ToName, e.ToFQN, e.ToKind = name, x.qualify(name), KindFunction
	case "selector_expression":
		field := x.text(x.field(fn, "field"))
		if field == "" {
			return
		}
		e.ToName, e.ToKind = field, KindMethod
		operand := x.field(fn, "operand")
		if x.typ(operand) == "identifier" {
			op := x.text(operand)
			if path, ok := x.imports[op]; ok {
				e.ToFQN, e.ToKind = path+"."+field, KindFunction
				break
			}
			if op == sc.recvName && sc.recvType != "" {
				e.ToFQN = x.qualify(sc.recvType + "." + field)
				break
			}
		}
		e.Confidence = ConfName
	default:
		return
	}
	x.edge(e)
}

func (x *goExtractor) literal(n *ts.Node, sc goScope) {
	t := x.field(n, "type")
	if x.typ(t) == "generic_type" {
		t = x.field(t, "type")
	}
	var target, name string
	switch x.typ(t) {
	case "type_identifier":
		name = x.text(t)
		target = x.qualify(name)
	case "qualified_type":
		name = x.text(x.field(t, "name"))
		path, ok := x.imports[x.text(x.field(t, "package"))]
		if !ok {
			return
		}
		target = path + "." + name
	default:
		return // slices, maps, anonymous structs
	}
	x.edge(parsedEdge{From: x.from(sc), Kind: EdgeInstantiates, ToName: name, ToFQN: target, ToKind: KindStruct, Line: lineStart(n)})
}
