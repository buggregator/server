package xdebug

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectTools(t *testing.T, p *Proxy, allowEval bool) *sdkmcp.ClientSession {
	t.Helper()
	server := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "test", Version: "1"}, nil)
	registerTools(server, p, allowEval)
	ct, st := sdkmcp.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "c", Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func callTool(t *testing.T, cs *sdkmcp.ClientSession, name string, args map[string]any) (string, bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdkmcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var sb strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String(), res.IsError
}

func b64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

var cmdNameRe = regexp.MustCompile(`^(\S+)`)

// serveEngine answers agent commands like Xdebug 3 does.
func serveEngine(t *testing.T, e *fakeEngine, stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			return
		default:
		}
		line, err := readCommand(e.r)
		if err != nil {
			return
		}
		tid := tidRe.FindStringSubmatch(line)[1]
		var inner string
		switch cmdNameRe.FindString(line) {
		case "stack_get":
			inner = `<stack where="App\User->email" level="0" type="file" filename="file:///app/src/User.php" lineno="12"></stack>` +
				`<stack where="{main}" level="1" type="file" filename="file:///app/public/index.php" lineno="5"></stack>`
		case "source":
			e.send(response("source", tid, ` encoding="base64"`, b64("line 7\nline 8\nline 9")))
			continue
		case "context_get":
			inner = `<property name="$id" fullname="$id" type="int"><![CDATA[42]]></property>` +
				`<property name="$name" fullname="$name" type="string" size="5" encoding="base64"><![CDATA[` + b64("Alice") + `]]></property>` +
				`<property name="$user" fullname="$user" type="object" classname="App\User" children="1" numchildren="2" page="0" pagesize="32">` +
				`<property name="email" fullname="$user-&gt;email" facet="private" type="string" size="9" encoding="base64"><![CDATA[` + b64("a@b.c.dev") + `]]></property>` +
				`<property name="tags" fullname="$user-&gt;tags" facet="public" type="array" children="1" numchildren="3"></property>` +
				`</property>` +
				`<property name="$none" fullname="$none" type="null"></property>`
		case "property_get":
			if !strings.Contains(line, `-n "$user->tags[\"a b\"]"`) && !strings.Contains(line, "-n $user->tags") {
				e.send(response("property_get", tid, "", `<error code="300"><message><![CDATA[can not get property]]></message></error>`))
				continue
			}
			inner = `<property name="tags" fullname="$user-&gt;tags" type="array" children="1" numchildren="1" page="0" pagesize="32">` +
				`<property name="0" fullname="$user-&gt;tags[0]" type="string" size="3" encoding="base64"><![CDATA[` + b64("vip") + `]]></property></property>`
		case "eval":
			expr, _ := base64.StdEncoding.DecodeString(strings.SplitN(line, " -- ", 2)[1])
			if string(expr) != "count($items)" {
				t.Errorf("eval expr %q", expr)
			}
			inner = `<property type="int"><![CDATA[3]]></property>`
		default:
			inner = `<error code="4"><message><![CDATA[unimplemented]]></message></error>`
		}
		e.send(response(cmdNameRe.FindString(line), tid, "", inner))
	}
}

func TestMCPTools_ReadPausedSession(t *testing.T) {
	h := newHarness(t)
	e, d, s := h.connect(t)
	cs := connectTools(t, h.proxy, true)

	// Not paused yet: tools refuse.
	if out, isErr := callTool(t, cs, "xdebug_stack", map[string]any{}); !isErr || !strings.Contains(out, "no paused") {
		t.Fatalf("stack before pause: %v %s", isErr, out)
	}

	pause(t, e, d, "1")
	stop := make(chan struct{})
	defer close(stop)
	go serveEngine(t, e, stop)

	out, isErr := callTool(t, cs, "xdebug_sessions", map[string]any{})
	if isErr || !strings.Contains(out, `"state": "break"`) || !strings.Contains(out, `"idekey": "PHPSTORM"`) {
		t.Fatalf("sessions: %s", out)
	}

	out, isErr = callTool(t, cs, "xdebug_location", map[string]any{"session_id": s.ID})
	if isErr || !strings.Contains(out, `"line": 12`) || !strings.Contains(out, `App\\User->email`) || !strings.Contains(out, "7: line 7") {
		t.Fatalf("location: %s", out)
	}

	out, isErr = callTool(t, cs, "xdebug_stack", map[string]any{})
	var frames []Frame
	if isErr || json.Unmarshal([]byte(out), &frames) != nil || len(frames) != 2 || frames[1].Where != "{main}" {
		t.Fatalf("stack: %s", out)
	}

	out, isErr = callTool(t, cs, "xdebug_variables", map[string]any{})
	var vars []Var
	if isErr || json.Unmarshal([]byte(out), &vars) != nil || len(vars) != 4 {
		t.Fatalf("variables: %s", out)
	}
	if *vars[0].Value != "42" || *vars[1].Value != "Alice" || vars[3].Value != nil {
		t.Fatalf("scalar values: %s", out)
	}
	user := vars[2]
	if user.Class != `App\User` || len(user.Children) != 2 || *user.Children[0].Value != "a@b.c.dev" ||
		user.Children[0].FullName != "$user->email" || user.Children[1].NumChildren != 3 {
		t.Fatalf("object: %s", out)
	}

	out, isErr = callTool(t, cs, "xdebug_property", map[string]any{"name": "$user->tags"})
	if isErr || !strings.Contains(out, `"value": "vip"`) {
		t.Fatalf("property: %s", out)
	}
	out, isErr = callTool(t, cs, "xdebug_property", map[string]any{"name": "$nope"})
	if !isErr || !strings.Contains(out, "can not get property") {
		t.Fatalf("missing property: %v %s", isErr, out)
	}

	out, isErr = callTool(t, cs, "xdebug_eval", map[string]any{"expression": "count($items)"})
	if isErr || !strings.Contains(out, `"value": "3"`) {
		t.Fatalf("eval: %s", out)
	}

	// None of this reached the IDE.
	d.expectQuiet(100 * time.Millisecond)
}

func TestMCPTools_EvalOffByDefault(t *testing.T) {
	cs := connectTools(t, NewProxy("127.0.0.1:1"), false)
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, tool := range res.Tools {
		names[tool.Name] = true
	}
	for _, want := range []string{"xdebug_sessions", "xdebug_location", "xdebug_stack", "xdebug_variables", "xdebug_property"} {
		if !names[want] {
			t.Errorf("missing %s", want)
		}
	}
	if names["xdebug_eval"] {
		t.Error("xdebug_eval registered with allow_eval off")
	}
	if out, isErr := callTool(t, cs, "xdebug_sessions", map[string]any{}); isErr || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty sessions: %s", out)
	}
}
