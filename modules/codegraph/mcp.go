package codegraph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func textResult(v any) *sdkmcp.CallToolResult {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: strings.TrimRight(buf.String(), "\n")}}}
}

type searchInput struct {
	Query string `json:"query" jsonschema:"Name or part of a full name: UserService, App\\Service\\User, Save, github.com/acme/app/repo.New"`
	Kind  string `json:"kind,omitempty" jsonschema:"Optional kinds, comma-separated: class, interface, trait, enum, struct, type, function, method, namespace, package"`
	Root  string `json:"root,omitempty" jsonschema:"Optional root name (code_index_status lists them)"`
	Limit int    `json:"limit,omitempty" jsonschema:"Max results (default 50)"`
}

type symbolInput struct {
	Symbol string `json:"symbol" jsonschema:"Symbol id from code_search, or a full name: App\\Service\\UserService::save, github.com/acme/app/svc.S.Save"`
	Root   string `json:"root,omitempty" jsonschema:"Optional root name, to pick among roots when a full name is given"`
}

type callsInput struct {
	Symbol string `json:"symbol" jsonschema:"Symbol id or full name"`
	Root   string `json:"root,omitempty" jsonschema:"Optional root name"`
	Depth  int    `json:"depth,omitempty" jsonschema:"Levels to follow, 1-5 (default 1)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"Max nodes (default 200)"`
}

type outlineInput struct {
	Root string `json:"root,omitempty" jsonschema:"Root name; may be left out when only one root is configured"`
	Path string `json:"path" jsonschema:"File path relative to the root, as code_search answers show it"`
}

type frameInput struct {
	File string `json:"file" jsonschema:"File path from a stack frame or profile: absolute (code_graph.path_map is applied) or relative to a root"`
	Line int    `json:"line" jsonschema:"Line number in that file"`
	Root string `json:"root,omitempty" jsonschema:"Optional root name"`
}

func registerTools(server *sdkmcp.Server, g *Graphs, ix *Indexer) {
	lookup := func(ref, root string) (SymbolRef, error) {
		s, err := g.Lookup(ref, root)
		if errors.Is(err, errNotFound) {
			return s, fmt.Errorf("symbol %q not found; use code_search to find its id", ref)
		}
		return s, err
	}

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_index_status",
		Description: "List the source folders (roots) whose code graph Buggregator keeps, with index state (pending, indexing, ready, error), file, symbol and edge counts, and when each was indexed. Only folders from the server config are indexed.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		return textResult(ix.Statuses()), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_search",
		Description: "Find classes, interfaces, traits, enums, structs, functions, methods, namespaces or packages in the indexed code by name or by part of the full name. Returns ids for code_symbol, code_callers, code_callees and code_dependencies.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in searchInput) (*sdkmcp.CallToolResult, any, error) {
		res, err := g.Search(in.Query, in.Kind, in.Root, in.Limit)
		if err != nil {
			return nil, nil, err
		}
		return textResult(res), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_symbol",
		Description: "Describe one symbol: file, lines, signature, parent, members, what it extends, implements, uses or imports, who extends or implements it, and counts of callers and callees. Includes its source when code_graph.snippets is on.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in symbolInput) (*sdkmcp.CallToolResult, any, error) {
		s, err := lookup(in.Symbol, in.Root)
		if err != nil {
			return nil, nil, err
		}
		d, err := g.Detail(s.ID)
		if err != nil {
			return nil, nil, err
		}
		return textResult(d), nil, nil
	})

	calls := func(incoming bool) func(ctx context.Context, req *sdkmcp.CallToolRequest, in callsInput) (*sdkmcp.CallToolResult, any, error) {
		return func(ctx context.Context, req *sdkmcp.CallToolRequest, in callsInput) (*sdkmcp.CallToolResult, any, error) {
			s, err := lookup(in.Symbol, in.Root)
			if err != nil {
				return nil, nil, err
			}
			gr, err := g.Calls(s.ID, incoming, in.Depth, in.Limit)
			if err != nil {
				return nil, nil, err
			}
			return textResult(gr), nil, nil
		}
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_callers",
		Description: "Who calls or instantiates a function, method or class, up to depth levels. Edges with confidence 'exact' are resolved from the source; 'name' edges are method calls on a value of unknown type that only match by method name and may be false. Node ids: s:<symbol id>, f:<file> for code outside any function.",
	}, calls(true))
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_callees",
		Description: "What a function or method calls or instantiates, up to depth levels. Targets outside the indexed code (vendor, standard library) are external nodes (x:<name>); 'name' edges are method calls whose receiver type is unknown.",
	}, calls(false))

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_file_outline",
		Description: "The structure of one source file: namespace or package, types, functions and methods with their lines, as a tree.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in outlineInput) (*sdkmcp.CallToolResult, any, error) {
		root := in.Root
		if root == "" && len(ix.Roots()) == 1 {
			root = ix.Roots()[0].Name
		}
		items, err := g.Outline(root, in.Path)
		if errors.Is(err, errNotFound) {
			return nil, nil, fmt.Errorf("file %q is not indexed in root %q", in.Path, root)
		}
		if err != nil {
			return nil, nil, err
		}
		return textResult(items), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_dependencies",
		Description: "Dependencies of a class, type, namespace or package: what it imports, extends, implements and uses, and the namespaces or packages it depends on and is depended on by (edge counts).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in symbolInput) (*sdkmcp.CallToolResult, any, error) {
		s, err := lookup(in.Symbol, in.Root)
		if err != nil {
			return nil, nil, err
		}
		d, err := g.Detail(s.ID)
		if err != nil {
			return nil, nil, err
		}
		gr, err := g.Subgraph(GraphQuery{Root: s.Root, Level: LevelPackage, Target: s.ID, Depth: 1})
		if err != nil {
			return nil, nil, err
		}
		return textResult(map[string]any{"symbol": d.SymbolRef, "relations": d.Relations, "incoming": d.Incoming, "packages": gr}), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "code_resolve_frame",
		Description: "Map a file and line from an error stack frame or a profile to the function or method in the indexed code, with its direct callers. Use it with sentry_event or profiler tools to see the code around a failing frame.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in frameInput) (*sdkmcp.CallToolResult, any, error) {
		s, err := g.Resolve(in.File, in.Line, in.Root)
		if errors.Is(err, errNotFound) {
			return nil, nil, fmt.Errorf("no indexed symbol at %s:%d (check code_graph.roots and code_graph.path_map)", in.File, in.Line)
		}
		if err != nil {
			return nil, nil, err
		}
		callers, err := g.Calls(s.ID, true, 1, 50)
		if err != nil {
			return nil, nil, err
		}
		return textResult(map[string]any{"symbol": s, "callers": callers}), nil, nil
	})
}
