package xdebug

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- DBGp response model -----------------------------------------------------

type encText struct {
	Encoding string `xml:"encoding,attr"`
	Text     string `xml:",chardata"`
}

func (e *encText) decode() string {
	if e == nil {
		return ""
	}
	return decodeText(e.Encoding, e.Text)
}

func decodeText(encoding, text string) string {
	if encoding == "base64" {
		if b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text)); err == nil {
			return string(b)
		}
	}
	return text
}

type xProperty struct {
	Name        string      `xml:"name,attr"`
	FullName    string      `xml:"fullname,attr"`
	Type        string      `xml:"type,attr"`
	ClassName   string      `xml:"classname,attr"`
	Facet       string      `xml:"facet,attr"`
	NumChildren int         `xml:"numchildren,attr"`
	Page        int         `xml:"page,attr"`
	PageSize    int         `xml:"pagesize,attr"`
	Encoding    string      `xml:"encoding,attr"`
	Size        int         `xml:"size,attr"`
	Text        string      `xml:",chardata"`
	NameEl      *encText    `xml:"name"`
	FullNameEl  *encText    `xml:"fullname"`
	ClassNameEl *encText    `xml:"classname"`
	ValueEl     *encText    `xml:"value"`
	Props       []xProperty `xml:"property"`
}

type xResponse struct {
	Command  string `xml:"command,attr"`
	Status   string `xml:"status,attr"`
	Encoding string `xml:"encoding,attr"`
	Text     string `xml:",chardata"`
	Error    *struct {
		Code    string `xml:"code,attr"`
		Message string `xml:"message"`
	} `xml:"error"`
	Stack []struct {
		Level    int    `xml:"level,attr"`
		Type     string `xml:"type,attr"`
		Filename string `xml:"filename,attr"`
		Lineno   int    `xml:"lineno,attr"`
		Where    string `xml:"where,attr"`
	} `xml:"stack"`
	Contexts []struct {
		Name string `xml:"name,attr"`
		ID   int    `xml:"id,attr"`
	} `xml:"context"`
	Props       []xProperty `xml:"property"`
	Breakpoints []struct {
		ID       string `xml:"id,attr"`
		Type     string `xml:"type,attr"`
		State    string `xml:"state,attr"`
		Filename string `xml:"filename,attr"`
		Lineno   int    `xml:"lineno,attr"`
		Function string `xml:"function,attr"`
		HitCount int    `xml:"hit_count,attr"`
		Resolved string `xml:"resolved,attr"`
	} `xml:"breakpoint"`
}

// --- Agent-facing model ------------------------------------------------------

// Var is a decoded variable, array element or object property.
type Var struct {
	Name        string  `json:"name"`
	FullName    string  `json:"fullname,omitempty"`
	Type        string  `json:"type"`
	Class       string  `json:"class,omitempty"`
	Facet       string  `json:"facet,omitempty"`
	Value       *string `json:"value,omitempty"`
	Size        int     `json:"size,omitempty"`
	NumChildren int     `json:"numchildren,omitempty"`
	Page        int     `json:"page,omitempty"`
	PageSize    int     `json:"pagesize,omitempty"`
	Children    []Var   `json:"children,omitempty"`
}

func toVar(p xProperty) Var {
	v := Var{
		Name: p.Name, FullName: p.FullName, Type: p.Type, Class: p.ClassName, Facet: p.Facet,
		Size: p.Size, NumChildren: p.NumChildren,
	}
	// extended_properties mode moves names and values into child elements.
	if p.NameEl != nil {
		v.Name = p.NameEl.decode()
	}
	if p.FullNameEl != nil {
		v.FullName = p.FullNameEl.decode()
	}
	if p.ClassNameEl != nil {
		v.Class = p.ClassNameEl.decode()
	}
	if len(p.Props) > 0 || p.NumChildren > 0 {
		v.Page, v.PageSize = p.Page, p.PageSize
		for _, c := range p.Props {
			v.Children = append(v.Children, toVar(c))
		}
	} else if p.Type != "null" && p.Type != "uninitialized" {
		var s string
		if p.ValueEl != nil {
			s = p.ValueEl.decode()
		} else {
			s = decodeText(p.Encoding, p.Text)
		}
		v.Value = &s
	}
	if v.Size == len(derefOr(v.Value)) {
		v.Size = 0 // only worth showing when the value was cut by max_data
	}
	return v
}

func derefOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Frame is one stack frame.
type Frame struct {
	Level int    `json:"level"`
	Where string `json:"where,omitempty"`
	File  string `json:"file"`
	Line  int    `json:"line"`
	Type  string `json:"type,omitempty"`
}

// --- Session helpers ---------------------------------------------------------

func (s *Session) call(name, args string, data []byte) (*xResponse, error) {
	body, err := s.Command(name, args, data)
	if err != nil {
		return nil, err
	}
	return parseResponse(body)
}

func parseResponse(body []byte) (*xResponse, error) {
	var r xResponse
	if err := unmarshalXML(body, &r); err != nil {
		return nil, fmt.Errorf("cannot parse Xdebug reply: %w", err)
	}
	if r.Error != nil {
		return nil, &dbgpError{Code: r.Error.Code, Message: strings.TrimSpace(r.Error.Message)}
	}
	return &r, nil
}

// dbgpError is an error Xdebug answered with (e.g. 300: can not get property).
type dbgpError struct {
	Code    string
	Message string
}

func (e *dbgpError) Error() string {
	return "xdebug error " + e.Code + ": " + e.Message
}

func (s *Session) stack() ([]Frame, error) {
	r, err := s.call("stack_get", "", nil)
	if err != nil {
		return nil, err
	}
	frames := make([]Frame, 0, len(r.Stack))
	for _, f := range r.Stack {
		if s.seen != nil {
			s.seen(f.Filename)
		}
		frames = append(frames, Frame{Level: f.Level, Where: f.Where, File: f.Filename, Line: f.Lineno, Type: f.Type})
	}
	return frames, nil
}

// sourceLines returns lines [from, to] of a file as "N: text" lines.
func (s *Session) sourceLines(file string, from, to int) (string, error) {
	if from < 1 {
		from = 1
	}
	r, err := s.call("source", fmt.Sprintf("-f %s -b %d -e %d", quoteArg(file), from, to), nil)
	if err != nil {
		return "", err
	}
	text := decodeText(r.Encoding, r.Text)
	var sb strings.Builder
	for i, l := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		fmt.Fprintf(&sb, "%d: %s\n", from+i, l)
	}
	return sb.String(), nil
}

// --- Tools ---------------------------------------------------------------------

type sessionInput struct {
	SessionID string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
}

type variablesInput struct {
	SessionID string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
	Depth     int    `json:"depth,omitempty" jsonschema:"Stack frame level (0 = current frame, see xdebug_stack)"`
	ContextID int    `json:"context_id,omitempty" jsonschema:"0 = local variables (default), 1 = superglobals, 2 = user-defined constants"`
}

type propertyInput struct {
	SessionID string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
	Name      string `json:"name" jsonschema:"Full name of the variable or property, as in 'fullname' of xdebug_variables (e.g. $user, $user->address, $items[0])"`
	Depth     int    `json:"depth,omitempty" jsonschema:"Stack frame level (0 = current frame)"`
	ContextID int    `json:"context_id,omitempty" jsonschema:"0 = locals (default), 1 = superglobals, 2 = constants"`
	Page      int    `json:"page,omitempty" jsonschema:"Page of children for large arrays/objects (0-based)"`
}

type evalInput struct {
	SessionID  string `json:"session_id,omitempty" jsonschema:"Session id from xdebug_sessions. May be omitted when exactly one session is paused."`
	Expression string `json:"expression" jsonschema:"PHP expression evaluated in the current frame, e.g. count($items) or $user->getEmail()"`
}

const maxPropertyData = 64 << 10

func textResult(v any) *sdkmcp.CallToolResult {
	// No HTML escaping: "->" and "<" in PHP names and values stay readable.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
	return &sdkmcp.CallToolResult{Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: strings.TrimRight(buf.String(), "\n")}}}
}

// runFunc resolves a session, logs the call in its agent log (UI) and renders
// the result.
type runFunc func(sessionID, tool, args string, fn func(s *Session) (any, string, error)) (*sdkmcp.CallToolResult, any, error)

func registerTools(server *sdkmcp.Server, p *Proxy, allowEval, allowControl bool) {
	// run resolves the session, logs the call in its agent log (UI) and
	// renders the result. Calls that never reach a session are not logged.
	var run runFunc = func(sessionID, tool, args string, fn func(s *Session) (any, string, error)) (*sdkmcp.CallToolResult, any, error) {
		s, err := p.Session(sessionID)
		if err != nil {
			return nil, nil, err
		}
		args = shortArgs(args)
		started := time.Now()
		s.agentStarted(tool, args)
		out, summary, err := fn(s)
		if err != nil {
			s.agentDone(tool, args, started, false, "refused — "+err.Error())
			return nil, nil, err
		}
		s.agentDone(tool, args, started, true, summary)
		return textResult(out), nil, nil
	}

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_sessions",
		Description: "List Xdebug debug sessions, including ones closed in the last 10 minutes. driver 'ide': an IDE drives the session and the xdebug_* tools only read it. driver 'here': no IDE, Buggregator drives it, and xdebug_step / xdebug_continue / xdebug_stop control it (when xdebug.allow_control is on). A session can be read only while its state is 'break'. xdebug_eval runs a PHP expression in the paused request and exists only when xdebug.allow_eval is true.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, _ struct{}) (*sdkmcp.CallToolResult, any, error) {
		infos := []Info{}
		for _, s := range p.Sessions() {
			infos = append(infos, s.Info())
		}
		return textResult(infos), nil, nil
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_location",
		Description: "Current file and line of a paused Xdebug session, the function it is in, and the source lines around it.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in sessionInput) (*sdkmcp.CallToolResult, any, error) {
		return run(in.SessionID, "xdebug_location", "", func(s *Session) (any, string, error) {
			frames, err := s.stack()
			if err != nil {
				return nil, "", err
			}
			if len(frames) == 0 {
				return nil, "", fmt.Errorf("xdebug returned an empty stack")
			}
			top := frames[0]
			out := map[string]any{"session_id": s.ID, "file": top.File, "line": top.Line, "where": top.Where}
			if src, err := s.sourceLines(top.File, top.Line-5, top.Line+5); err == nil {
				out["source"] = src
			}
			return out, fmt.Sprintf("%s:%d", baseName(top.File), top.Line), nil
		})
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_stack",
		Description: "Call stack of a paused Xdebug session: one frame per level, 0 is the current one.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in sessionInput) (*sdkmcp.CallToolResult, any, error) {
		return run(in.SessionID, "xdebug_stack", "", func(s *Session) (any, string, error) {
			frames, err := s.stack()
			if err != nil {
				return nil, "", err
			}
			return frames, plural(len(frames), "frame"), nil
		})
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_variables",
		Description: "Variables of a stack frame in a paused Xdebug session (locals by default). Arrays and objects show their first level; use xdebug_property with the 'fullname' to go deeper.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in variablesInput) (*sdkmcp.CallToolResult, any, error) {
		args := fmt.Sprintf("-d %d -c %d", in.Depth, in.ContextID)
		return run(in.SessionID, "xdebug_variables", args, func(s *Session) (any, string, error) {
			r, err := s.call("context_get", args, nil)
			if err != nil {
				return nil, "", err
			}
			vars := []Var{}
			for _, pr := range r.Props {
				vars = append(vars, toVar(pr))
			}
			return vars, plural(len(vars), "variable"), nil
		})
	})

	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_property",
		Description: "One variable, array element or object property of a paused Xdebug session, with its children (object properties, array items) and long string values in full (up to 64 KB).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in propertyInput) (*sdkmcp.CallToolResult, any, error) {
		if strings.TrimSpace(in.Name) == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		return run(in.SessionID, "xdebug_property", in.Name, func(s *Session) (any, string, error) {
			args := fmt.Sprintf("-d %d -c %d -m %d -p %d -n %s", in.Depth, in.ContextID, maxPropertyData, in.Page, quoteArg(in.Name))
			r, err := s.call("property_get", args, nil)
			if err != nil {
				return nil, "", err
			}
			if len(r.Props) == 0 {
				return nil, "", fmt.Errorf("property %s not found", in.Name)
			}
			v := toVar(r.Props[0])
			return v, varSummary(v), nil
		})
	})

	if allowControl {
		registerControlTools(server, p, run)
	}
	registerBreakpointTools(server, p, allowControl)
	if !allowEval {
		return
	}
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name:        "xdebug_eval",
		Description: "Evaluate a PHP expression in the current frame of a paused Xdebug session and return its value. Use expressions without side effects: the code runs inside the debugged request.",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in evalInput) (*sdkmcp.CallToolResult, any, error) {
		if strings.TrimSpace(in.Expression) == "" {
			return nil, nil, fmt.Errorf("expression is required")
		}
		return run(in.SessionID, "xdebug_eval", in.Expression, func(s *Session) (any, string, error) {
			v, err := s.eval(in.Expression)
			if err != nil {
				return nil, "", err
			}
			return v, varSummary(v), nil
		})
	})
}

// eval evaluates an expression; an empty reply is null.
func (s *Session) eval(expr string) (Var, error) {
	r, err := s.call("eval", "", []byte(expr))
	if err != nil {
		return Var{}, err
	}
	if len(r.Props) == 0 {
		return Var{Type: "null"}, nil
	}
	return toVar(r.Props[0]), nil
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// varSummary is a short line for the agent log: "string(17)", "array(3)", "User".
func varSummary(v Var) string {
	switch {
	case v.Class != "":
		return v.Class
	case v.NumChildren > 0 || v.Type == "array":
		return fmt.Sprintf("%s(%d)", v.Type, v.NumChildren)
	case v.Value != nil && len(*v.Value) <= 40:
		return v.Type + " " + *v.Value
	default:
		return v.Type
	}
}

func baseName(file string) string {
	if i := strings.LastIndexByte(file, '/'); i >= 0 {
		return file[i+1:]
	}
	return file
}
