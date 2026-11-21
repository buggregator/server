package codegraph

import (
	"path/filepath"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	golang "github.com/odvcencio/gotreesitter/grammars/go"
	"github.com/odvcencio/gotreesitter/grammars/php"
)

// Symbol kinds.
const (
	KindNamespace = "namespace" // PHP namespace (one row per file that declares it)
	KindPackage   = "package"   // Go package (one row per file)
	KindClass     = "class"
	KindInterface = "interface"
	KindTrait     = "trait"
	KindEnum      = "enum"
	KindStruct    = "struct" // Go struct type
	KindType      = "type"   // other Go named type
	KindFunction  = "function"
	KindMethod    = "method"
)

// Edge kinds.
const (
	EdgeImports      = "imports"
	EdgeExtends      = "extends"
	EdgeImplements   = "implements"
	EdgeUses         = "uses" // PHP trait use
	EdgeCalls        = "calls"
	EdgeInstantiates = "instantiates"
)

// Edge confidence: exact = the target name is known (it may still be outside
// the indexed code); name = only the method name is known, the receiver type
// is not, so the edge matches every method with that name.
const (
	ConfExact = "exact"
	ConfName  = "name"
)

// parsedSymbol is one declaration found in a file. Parent is the index of the
// enclosing symbol in the same file, -1 for none.
type parsedSymbol struct {
	Kind      string
	Name      string
	FQN       string
	Parent    int
	LineStart int
	LineEnd   int
	Signature string
}

// parsedEdge is one reference found in a file. From is the index of the
// enclosing symbol (-1: the file's namespace/package symbol, if any).
type parsedEdge struct {
	From       int
	Kind       string
	ToName     string // short name: method or function or class name
	ToFQN      string // full name when known
	AltFQN     string // second candidate (PHP global function fallback)
	ToKind     string // expected target kind hint: method, function, class
	Line       int
	Confidence string
}

type parsedFile struct {
	Symbols  []parsedSymbol
	Edges    []parsedEdge
	HasError bool
}

// language detects the language of a file by its extension.
func language(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".php":
		return "php"
	case ".go":
		return "go"
	}
	return ""
}

// parsers holds one tree-sitter parser per language. A parser is not safe for
// concurrent use; an indexing run owns its own set.
type parsers struct {
	php, golang *ts.Parser
}

func newParsers() *parsers {
	return &parsers{php: ts.NewParser(php.Language()), golang: ts.NewParser(golang.Language())}
}

// parse extracts symbols and edges. goPkg is the Go import path of the file's
// directory (ignored for PHP).
func (p *parsers) parse(lang string, src []byte, goPkg string) (*parsedFile, error) {
	switch lang {
	case "php":
		tree, err := p.php.Parse(src)
		if err != nil {
			return nil, err
		}
		defer tree.Release()
		return extractPHP(tree.RootNode(), src, php.Language()), nil
	case "go":
		tree, err := p.golang.Parse(src)
		if err != nil {
			return nil, err
		}
		defer tree.Release()
		return extractGo(tree.RootNode(), src, golang.Language(), goPkg), nil
	}
	return &parsedFile{}, nil
}

// node helpers shared by the extractors.

type nodeReader struct {
	src  []byte
	lang *ts.Language
}

func (r nodeReader) typ(n *ts.Node) string {
	if n == nil {
		return ""
	}
	return n.Type(r.lang)
}

func (r nodeReader) text(n *ts.Node) string {
	if n == nil {
		return ""
	}
	return n.Text(r.src)
}

func (r nodeReader) field(n *ts.Node, name string) *ts.Node {
	if n == nil {
		return nil
	}
	return n.ChildByFieldName(name, r.lang)
}

// namedChildren returns the named children of n.
func namedChildren(n *ts.Node) []*ts.Node {
	if n == nil {
		return nil
	}
	out := make([]*ts.Node, 0, n.NamedChildCount())
	for i := 0; i < n.NamedChildCount(); i++ {
		out = append(out, n.NamedChild(i))
	}
	return out
}

func lineStart(n *ts.Node) int { return int(n.StartPoint().Row) + 1 }
func lineEnd(n *ts.Node) int   { return int(n.EndPoint().Row) + 1 }
