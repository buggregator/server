package httpdumps

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestMockCondition_Holds(t *testing.T) {
	body := `{"card":{"token":"tok_ok","cvc":123,"saved":true,"note":null,"meta":{"a":1,"b":[1,2]}},
		"items":[{"sku":"A-1","qty":2},{"sku":"B-2","qty":1.5}],"total":15,"id":9007199254740993}`
	req := &mockRequest{
		Header: http.Header{"Authorization": {"Bearer good"}, "X-Tag": {"one", "two"}},
		Body:   []byte(body),
	}
	tests := []struct {
		name string
		c    MockCondition
		want bool
	}{
		{"header equals", MockCondition{In: "header", Name: "Authorization", Op: "equals", Value: "Bearer good"}, true},
		{"header name any case", MockCondition{In: "header", Name: "authorization", Op: "equals", Value: "Bearer good"}, true},
		{"header value is case-sensitive", MockCondition{In: "header", Name: "Authorization", Op: "equals", Value: "bearer good"}, false},
		{"header any of several values", MockCondition{In: "header", Name: "x-tag", Op: "equals", Value: "two"}, true},
		{"header not_equals", MockCondition{In: "header", Name: "Authorization", Op: "not_equals", Value: "Bearer bad"}, true},
		{"missing header never equals", MockCondition{In: "header", Name: "X-None", Op: "equals", Value: ""}, false},
		{"missing header is not_equals", MockCondition{In: "header", Name: "X-None", Op: "not_equals", Value: "x"}, true},
		{"header exists", MockCondition{In: "header", Name: "x-tag", Op: "exists"}, true},
		{"header not_exists", MockCondition{In: "header", Name: "X-None", Op: "not_exists"}, true},

		{"nested string", MockCondition{In: "body", Name: "card.token", Op: "equals", Value: "tok_ok"}, true},
		{"jsonpath form", MockCondition{In: "body", Name: "$.card.token", Op: "equals", Value: "tok_ok"}, true},
		{"number as number", MockCondition{In: "body", Name: "card.cvc", Op: "equals", Value: "123.0"}, true},
		{"number vs text", MockCondition{In: "body", Name: "card.cvc", Op: "equals", Value: "abc"}, false},
		{"bool", MockCondition{In: "body", Name: "card.saved", Op: "equals", Value: "true"}, true},
		{"null", MockCondition{In: "body", Name: "card.note", Op: "equals", Value: "null"}, true},
		{"null exists", MockCondition{In: "body", Name: "card.note", Op: "exists"}, true},
		{"object as JSON", MockCondition{In: "body", Name: "card.meta", Op: "equals", Value: `{"b":[1,2],"a":1.0}`}, true},
		{"list index", MockCondition{In: "body", Name: "items.1.sku", Op: "equals", Value: "B-2"}, true},
		{"bracket index", MockCondition{In: "body", Name: "$.items[0].qty", Op: "equals", Value: "2"}, true},
		{"any item", MockCondition{In: "body", Name: "items.*.sku", Op: "equals", Value: "B-2"}, true},
		{"any item, none", MockCondition{In: "body", Name: "items.*.sku", Op: "equals", Value: "C-3"}, false},
		{"not_equals over any item", MockCondition{In: "body", Name: "items.*.sku", Op: "not_equals", Value: "A-1"}, false},
		{"index out of range", MockCondition{In: "body", Name: "items.5.sku", Op: "exists"}, false},
		{"missing field never equals", MockCondition{In: "body", Name: "card.number", Op: "equals", Value: ""}, false},
		{"missing field is not_equals", MockCondition{In: "body", Name: "card.number", Op: "not_equals", Value: "4242"}, true},
		{"missing field not_exists", MockCondition{In: "body", Name: "card.number", Op: "not_exists"}, true},
		{"top-level", MockCondition{In: "body", Name: "total", Op: "not_equals", Value: "15"}, false},
		{"big integer exact", MockCondition{In: "body", Name: "id", Op: "equals", Value: "9007199254740993"}, true},
		{"big integer off by one", MockCondition{In: "body", Name: "id", Op: "equals", Value: "9007199254740992"}, false},
		{"big integer inside an object", MockCondition{In: "body", Name: "card.meta", Op: "equals", Value: `{"a":1,"b":[1,2]}`}, true},
		{"fraction as float", MockCondition{In: "body", Name: "items.1.qty", Op: "equals", Value: "1.50"}, true},
		{"exponent as float", MockCondition{In: "body", Name: "total", Op: "equals", Value: "1.5e1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.holds(req); got != tt.want {
				t.Errorf("holds = %v, want %v", got, tt.want)
			}
		})
	}

	// Not JSON: every body field is missing.
	plain := &mockRequest{Body: []byte("token=tok_ok")}
	for op, want := range map[string]bool{"equals": false, "exists": false, "not_equals": true, "not_exists": true} {
		if got := (MockCondition{In: "body", Name: "token", Op: op, Value: "tok_ok"}).holds(plain); got != want {
			t.Errorf("form body, %s = %v, want %v", op, got, want)
		}
	}
}

func TestMockRuleNormalize_Conditions(t *testing.T) {
	tests := []struct {
		name string
		cs   []MockCondition
		errs map[string]string // key → part of the message
	}{
		{"ok", []MockCondition{{In: "Header", Name: " X-A ", Value: "1"}, {In: "body", Name: "a.0.b", Op: "EXISTS", Value: "dropped"}}, nil},
		{"bad in", []MockCondition{{In: "query", Name: "a"}}, map[string]string{"conditions.0": "in must be"}},
		{"no name", []MockCondition{{In: "body", Op: "exists"}}, map[string]string{"conditions.0": "name is required"}},
		{"bad header", []MockCondition{{In: "header", Name: "a b"}}, map[string]string{"conditions.0": "invalid header name"}},
		{"bad op", []MockCondition{{In: "body", Name: "a", Op: "contains"}}, map[string]string{"conditions.0": "op must be"}},
		{"bad path", []MockCondition{{In: "body", Name: "a..b"}}, map[string]string{"conditions.0": "bad body path"}},
		{"long value", []MockCondition{{In: "body", Name: "a", Value: strings.Repeat("x", 4097)}}, map[string]string{"conditions.0": "at most 4096"}},
		{"second only", []MockCondition{{In: "body", Name: "a"}, {In: "body", Name: ""}}, map[string]string{"conditions.1": "name is required"}},
		{"too many", make([]MockCondition, 21), map[string]string{"conditions": "at most 20"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := MockRule{Path: "/pay", Conditions: tt.cs}
			err := r.Normalize()
			if tt.errs == nil {
				if err != nil {
					t.Fatal(err)
				}
				c := r.Conditions
				if c[0].In != "header" || c[0].Name != "X-A" || c[0].Op != "equals" || c[1].Op != "exists" || c[1].Value != "" {
					t.Errorf("normalized = %+v", c)
				}
				return
			}
			ve, _ := err.(ValidationError)
			for k, part := range tt.errs {
				if !strings.Contains(ve[k], part) {
					t.Errorf("errors = %v, want %s: …%s…", ve, k, part)
				}
			}
		})
	}
	r := MockRule{Path: "/x"}
	if err := r.Normalize(); err != nil || r.Conditions == nil || len(r.Conditions) != 0 {
		t.Errorf("no conditions: %v %#v", err, r.Conditions)
	}
}

// The issue's case: one endpoint, two answers by a body field and a header;
// a rule without conditions below catches the rest.
func TestMockServer_Conditions(t *testing.T) {
	env := newMockEnv(t, []MockRule{{Name: "yaml ok", Method: "POST", Path: "/yaml",
		Conditions: []MockCondition{{In: "body", Name: "id", Value: "1"}}, Body: "yaml"}})
	env.create(t, `{"name":"paid","method":"POST","path":"/payments","status":200,"body":"paid",
		"conditions":[{"in":"body","name":"card.token","op":"equals","value":"tok_ok"},{"in":"header","name":"authorization","op":"equals","value":"Bearer good"}]}`)
	env.create(t, `{"name":"unauthorized","method":"POST","path":"/payments","status":401,"body":"who",
		"conditions":[{"in":"header","name":"Authorization","op":"not_equals","value":"Bearer good"}]}`)
	env.create(t, `{"name":"declined","method":"POST","path":"/payments","status":402,"body":"declined"}`)

	good := map[string]string{"Authorization": "Bearer good"}
	tests := []struct {
		name, path, body string
		hdr              map[string]string
		status           int
		resp             string
	}{
		{"all conditions hold", "/mock/payments", `{"card":{"token":"tok_ok"}}`, good, 200, "paid"},
		{"body field differs", "/mock/payments", `{"card":{"token":"tok_declined"}}`, good, 402, "declined"},
		{"no header", "/mock/payments", `{"card":{"token":"tok_ok"}}`, nil, 401, "who"},
		{"not JSON", "/mock/payments", `card=tok_ok`, good, 402, "declined"},
		{"config rule condition", "/mock/yaml", `{"id":1}`, nil, 200, "yaml"},
		{"config rule condition fails", "/mock/yaml", `{"id":2}`, nil, 404, `{"error":"no mock rule matched POST /yaml"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := env.do(t, "POST", tt.path, tt.body, tt.hdr)
			if w.Code != tt.status || strings.TrimSpace(w.Body.String()) != tt.resp {
				t.Errorf("got %d %q, want %d %q", w.Code, w.Body.String(), tt.status, tt.resp)
			}
		})
	}

	// The captured request keeps its whole body after the conditions read it.
	for _, ev := range env.requests(t, "") {
		req := ev["payload"].(map[string]any)["request"].(map[string]any)
		if req["uri"] == "mock/payments" && req["body"] == "" {
			t.Errorf("captured request lost its body: %v", req)
		}
	}

	// "Why no rule matched" names the failed condition.
	env.do(t, "POST", "/mock/yaml", `{"id":3}`, nil)
	info := mockField(t, env.requests(t, "")[0])
	near, _ := json.Marshal(info["near"])
	if !strings.Contains(string(near), `"detail":"body id equals","reason":"conditions"`) {
		t.Errorf("near = %s", near)
	}
}

func TestMockAPI_Conditions(t *testing.T) {
	env := newMockEnv(t, nil)
	r := env.create(t, `{"name":"c","method":"POST","path":"/c","conditions":[{"in":"body","name":"a.b","value":"1"}]}`)
	if len(r.Conditions) != 1 || r.Conditions[0].Op != "equals" {
		t.Fatalf("created conditions = %+v", r.Conditions)
	}
	plain := env.create(t, `{"name":"plain","path":"/p"}`)
	if plain.Conditions == nil || len(plain.Conditions) != 0 {
		t.Errorf("rule without conditions: %#v, want []", plain.Conditions)
	}

	// An older client that sends no conditions keeps them.
	w := env.do(t, "PUT", "/api/mocks/"+r.ID, `{"name":"renamed","method":"POST","path":"/c"}`, nil)
	var got MockRule
	json.Unmarshal(w.Body.Bytes(), &got)
	if w.Code != 200 || len(got.Conditions) != 1 {
		t.Fatalf("update without conditions: %d %s", w.Code, w.Body.String())
	}
	// [] clears them; the list survives a reload from SQLite.
	env.do(t, "PUT", "/api/mocks/"+r.ID, `{"name":"renamed","method":"POST","path":"/c","conditions":[]}`, nil)
	w = env.do(t, "GET", "/api/mocks/"+r.ID, "", nil)
	json.Unmarshal(w.Body.Bytes(), &got)
	if len(got.Conditions) != 0 || !strings.Contains(w.Body.String(), `"conditions":[]`) {
		t.Errorf("cleared: %s", w.Body.String())
	}

	w = env.do(t, "POST", "/api/mocks", `{"path":"/x","conditions":[{"in":"body","name":"a","op":"like"}]}`, nil)
	if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusBadRequest {
		t.Errorf("invalid condition: %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "conditions.0") {
		t.Errorf("invalid condition body: %s", w.Body.String())
	}

	// The preview tells whether the sample request meets the conditions.
	for _, tt := range []struct {
		body    string
		matched bool
		failed  string
	}{{`{"a":{"b":1}}`, true, ""}, {`{"a":{"b":2}}`, false, "body a.b equals"}} {
		in := `{"rule":{"method":"POST","path":"/c","conditions":[{"in":"body","name":"a.b","value":"1"}]},"request":{"method":"POST","path":"/c","body":` + strconvQuote(tt.body) + `}}`
		w := env.do(t, "POST", "/api/mocks/preview", in, nil)
		var out map[string]any
		json.Unmarshal(w.Body.Bytes(), &out)
		if out["matched"] != tt.matched || (tt.failed != "" && out["failed_condition"] != tt.failed) {
			t.Errorf("preview %s: %s", tt.body, w.Body.String())
		}
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
