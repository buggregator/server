package explain

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

func TestEventBody_Types(t *testing.T) {
	tests := []struct {
		name string
		ev   event.Event
		want []string
		gone []string
	}{
		{"sentry", event.Event{Type: "sentry", Payload: sdkEvent(t, "node/02-event.json")},
			[]string{"Error: ", "Order 15 not found", "Stack (most recent call first)"}, nil},
		{"monolog", event.Event{Type: "monolog", Payload: json.RawMessage(`{"message":"Payment failed","channel":"app","level_name":"ERROR","datetime":"2026-11-25","context":{"order":15},"extra":{}}`)},
			[]string{"Log: app.ERROR", "Message: Payment failed", `Context: {"order":15}`}, []string{"Extra"}},
		{"http-dump", event.Event{Type: "http-dump", Payload: json.RawMessage(`{"host":"api.test","request":{"method":"POST","uri":"/orders?x=1","headers":{"Content-Type":["application/json"],"Authorization":["Bearer abcdefgh12345678"]},"body":"{\"id\":15}"}}`)},
			[]string{"Request: POST api.test/orders?x=1", "Content-Type: application/json", `{"id":15}`}, nil},
		{"proxy dump with response", event.Event{Type: "http-dump", Payload: json.RawMessage(`{"host":"api.test","proxy":true,"duration_ms":120.4,"request":{"method":"GET","uri":"/"},"response":{"status_code":502,"headers":{},"body":"bad gateway"}}`)},
			[]string{"Response: 502 in 120 ms", "bad gateway"}, nil},
		{"var-dump html", event.Event{Type: "var-dump", Payload: json.RawMessage(`{"payload":{"type":"array","label":"cart","value":"<pre class=sf-dump>array:1 [<span>&quot;id&quot; =&gt; 15</span>]</pre>"},"context":{"file":"app/Cart.php","line":12}}`)},
			[]string{"Dump: cart (array)", `array:1 ["id" => 15]`, "app/Cart.php"}, []string{"<span>", "&quot;"}},
		{"profiler", event.Event{Type: "profiler", Payload: json.RawMessage(`{"app_name":"shop","hostname":"web-1","peaks":{"wt":250000,"cpu":200000,"mu":2048000,"pmu":4096000,"ct":10},
			"edges":{"e1":{"callee":"main()","cost":{"wt":250000,"ct":1},"diff":{"d_wt":10000}},"e2":{"callee":"PDO::execute","cost":{"wt":180000,"ct":3},"diff":{"d_wt":180000}},"e3":{"callee":"PDO::execute","cost":{"wt":20000,"ct":1},"diff":{"d_wt":20000}}}}`)},
			[]string{"App: shop", "Totals: wall 250.0 ms", "PDO::execute — own 200.0 ms, with callees 200.0 ms, calls 4", "main() — own 10.0 ms"}, nil},
		{"inspector", event.Event{Type: "inspector", Payload: json.RawMessage(`[{"model":"transaction","name":"GET /orders","type":"request","result":"200","duration":340.5,"host":{"hostname":"web-1"}},
			{"model":"segment","type":"mysql","label":"select * from orders","duration":250},{"model":"segment","type":"view","label":"orders.index","duration":40},
			{"model":"error","class":"RuntimeException","message":"boom","file":"app/X.php","line":7}]`)},
			[]string{"Transaction: GET /orders (request), result 200, 340.5 ms", "Error: RuntimeException: boom at app/X.php:7", "250.0 ms — mysql: select * from orders"}, nil},
		{"other type", event.Event{Type: "ray", Payload: json.RawMessage(`{"payloads":[{"type":"log","content":{"values":["hi"]}}]}`)},
			[]string{"Type: ray", `"values":["hi"]`}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := eventBody(tt.ev)
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range tt.want {
				if !strings.Contains(body, w) {
					t.Errorf("lacks %q:\n%s", w, body)
				}
			}
			for _, g := range tt.gone {
				if strings.Contains(body, g) {
					t.Errorf("has %q:\n%s", g, body)
				}
			}
		})
	}
	// The profiler lists the slowest first.
	body, _ := eventBody(tests[5].ev)
	if strings.Index(body, "PDO::execute") > strings.Index(body, "main()") {
		t.Errorf("profiler order:\n%s", body)
	}
}

// A big event is cut, never the question or the pointers; the node's lower
// limit wins over ours.
func TestContextText_Limits(t *testing.T) {
	big := event.Event{UUID: "u-1", Type: "ray", Project: "shop", Payload: json.RawMessage(`{"s":"` + strings.Repeat("a", 50000) + `"}`)}
	question := strings.Repeat("why ", 400)
	for _, limit := range []int{0, 16384, 3000} {
		text, err := contextText(Context{Kind: KindEvent, Key: "u-1"}, &big, question, pointer{MCP: true}, limit)
		if err != nil {
			t.Fatal(err)
		}
		max := itemTextBytes
		if limit > 0 && limit < max {
			max = limit
		}
		if len(text) > max {
			t.Errorf("limit %d: %d bytes", limit, len(text))
		}
		for _, w := range []string{"Question: " + strings.TrimSpace(question), "[cut to fit]", "Event: u-1 (ray, project shop)", "MCP connection (read-only): event u-1, project shop."} {
			if !strings.Contains(text, w) {
				t.Errorf("limit %d: lacks %.60q", limit, w)
			}
		}
	}
}

// The question, the page title and the page key are masked like the body.
func TestContextText_ScrubsAll(t *testing.T) {
	c := Context{Kind: KindPage, Key: "/p?token=abcdef123456", Title: "login api_key=k-123456", Text: "list DB_PASSWORD=hunter2"}
	text, err := contextText(c, nil, "my DB_PASSWORD=hunter2 please", pointer{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"hunter2", "abcdef123456", "k-123456"} {
		if strings.Contains(text, bad) {
			t.Errorf("%q sent:\n%s", bad, text)
		}
	}
	for _, want := range []string{"Question: my DB_PASSWORD=[redacted] please", "Page: /p?token=[redacted]"} {
		if !strings.Contains(text, want) {
			t.Errorf("lacks %q:\n%s", want, text)
		}
	}
}
