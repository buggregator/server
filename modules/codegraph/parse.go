package codegraph

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

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
	Stopped  string // timeout or cancelled: error recovery did not finish
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

// parseTimeout bounds one file's parse. tree-sitter's error recovery can run
// for minutes on an ordinary syntax error (a dropped quote in a 60 KB file);
// a parse that stops early marks the file parse_error and keeps no symbols.
const parseTimeout = 2 * time.Second

var errParseStopped = errors.New("parse stopped early")

// parsers holds one tree-sitter parser per language. A parser is not safe for
// concurrent use; an indexing run owns its own set.
type parsers struct {
	php, golang *ts.Parser
	cancel      uint32 // set to 1 to stop the parse in progress
}

func newParsers(timeout time.Duration) *parsers {
	p := &parsers{php: ts.NewParser(php.Language()), golang: ts.NewParser(golang.Language())}
	for _, pr := range []*ts.Parser{p.php, p.golang} {
		pr.SetTimeoutMicros(uint64(timeout.Microseconds()))
		pr.SetCancellationFlag(&p.cancel)
	}
	return p
}

// watch stops parsing when ctx ends; call the returned func when done.
func (p *parsers) watch(ctx context.Context) func() {
	stop := context.AfterFunc(ctx, func() { atomic.StoreUint32(&p.cancel, 1) })
	return func() { stop() }
}

// parse extracts symbols and edges. goPkg is the Go import path of the file's
// directory (ignored for PHP). A parse stopped by the timeout or the
// cancellation flag before any tree returns errParseStopped; one stopped in
// error recovery returns the first-pass result with Stopped set.
func (p *parsers) parse(lang string, src []byte, goPkg string) (*parsedFile, error) {
	var parser *ts.Parser
	var l *ts.Language
	switch lang {
	case "php":
		parser, l = p.php, php.Language()
	case "go":
		parser, l = p.golang, golang.Language()
	default:
		return &parsedFile{}, nil
	}
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, err
	}
	defer tree.Release()
	if tree.ParseStoppedEarly() {
		return nil, fmt.Errorf("%w: %v", errParseStopped, tree.ParseStopReason())
	}
	var out *parsedFile
	if lang == "php" {
		out = extractPHP(tree.RootNode(), src, l)
	} else {
		out = extractGo(tree.RootNode(), src, l, goPkg)
	}
	// The slow part is error recovery after the first pass: stopped there,
	// the tree of the first pass is kept and the file counts as broken.
	if r := tree.RetryStopReason(); r != ts.ParseStopNone {
		out.HasError, out.Stopped = true, string(r)
	}
	return out, nil
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
