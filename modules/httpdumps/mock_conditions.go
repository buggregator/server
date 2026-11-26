package httpdumps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

// Mock rule conditions (#512): a rule may also require request headers and
// JSON body fields. All conditions must hold, besides method, path and
// project; a rule whose conditions fail is skipped.

const (
	MaxMockConditions     = 20
	maxMockConditionValue = 4096
	maxMockConditionName  = 512
)

// Where a condition looks.
const (
	MockInHeader = "header"
	MockInBody   = "body"
)

// Comparisons. A missing field never equals: equals and exists fail on it,
// not_equals and not_exists hold.
const (
	MockOpEquals    = "equals"
	MockOpNotEquals = "not_equals"
	MockOpExists    = "exists"
	MockOpNotExists = "not_exists"
)

// MockCondition is one requirement on the request. Name is a header name
// (any case) or a JSON body path: card.number, items.0.id, items.*.id (any
// item); $.card.number and items[0].id are read the same.
type MockCondition struct {
	In    string `json:"in"`
	Name  string `json:"name"`
	Op    string `json:"op"`
	Value string `json:"value,omitempty"`
}

// String names the condition without its value, for "why no rule matched".
func (c MockCondition) String() string {
	return c.In + " " + c.Name + " " + c.Op
}

// normalizeConditions fills defaults and records invalid conditions in errs
// under "conditions" or "conditions.N".
func normalizeConditions(cs []MockCondition, errs ValidationError) []MockCondition {
	if cs == nil {
		return []MockCondition{}
	}
	if len(cs) > MaxMockConditions {
		errs["conditions"] = fmt.Sprintf("at most %d conditions", MaxMockConditions)
		return cs
	}
	for i := range cs {
		c := &cs[i]
		key := fmt.Sprintf("conditions.%d", i)
		c.In = strings.ToLower(strings.TrimSpace(c.In))
		c.Op = strings.ToLower(strings.TrimSpace(c.Op))
		c.Name = strings.TrimSpace(c.Name)
		if c.Op == "" {
			c.Op = MockOpEquals
		}
		switch {
		case c.In != MockInHeader && c.In != MockInBody:
			errs[key] = `in must be "header" or "body"`
		case c.Name == "":
			errs[key] = "name is required"
		case len(c.Name) > maxMockConditionName:
			errs[key] = fmt.Sprintf("name must be at most %d bytes", maxMockConditionName)
		case c.In == MockInHeader && strings.ContainsAny(c.Name, " :\r\n"):
			errs[key] = fmt.Sprintf("invalid header name %q", c.Name)
		case c.Op != MockOpEquals && c.Op != MockOpNotEquals && c.Op != MockOpExists && c.Op != MockOpNotExists:
			errs[key] = "op must be equals, not_equals, exists or not_exists"
		case len(c.Value) > maxMockConditionValue:
			errs[key] = fmt.Sprintf("value must be at most %d bytes", maxMockConditionValue)
		case c.In == MockInBody && bodyPath(c.Name) == nil:
			errs[key] = fmt.Sprintf("bad body path %q, use a.b, items.0.id or items.*.id", c.Name)
		}
		if c.Op == MockOpExists || c.Op == MockOpNotExists {
			c.Value = ""
		}
	}
	return cs
}

// bodyPath splits a body path into keys; nil when it is not one.
// "$.items[0].id", "items.0.id" and "items[0].id" are all [items 0 id].
func bodyPath(name string) []string {
	s := strings.TrimPrefix(strings.TrimPrefix(name, "$"), ".")
	s = strings.NewReplacer("[", ".", "]", "").Replace(s)
	if s == "" {
		return nil
	}
	keys := strings.Split(s, ".")
	for _, k := range keys {
		if k == "" {
			return nil
		}
	}
	return keys
}

// conditionsHold reports whether every condition of the rule holds for req,
// and the first that does not.
func (r *MockRule) conditionsHold(req *mockRequest) (bool, *MockCondition) {
	for i := range r.Conditions {
		if !r.Conditions[i].holds(req) {
			return false, &r.Conditions[i]
		}
	}
	return true, nil
}

func (c MockCondition) holds(req *mockRequest) bool {
	var found []any
	switch c.In {
	case MockInHeader:
		if req != nil {
			for _, v := range req.Header.Values(http.CanonicalHeaderKey(c.Name)) {
				found = append(found, v)
			}
		}
	case MockInBody:
		if req != nil {
			if v, ok := req.jsonNumbers(); ok {
				found = lookupBody(v, bodyPath(c.Name))
			}
		}
	}
	switch c.Op {
	case MockOpExists:
		return len(found) > 0
	case MockOpNotExists:
		return len(found) == 0
	}
	equal := false
	for _, v := range found {
		if jsonEquals(v, c.Value) {
			equal = true
			break
		}
	}
	if c.Op == MockOpNotEquals {
		return !equal
	}
	return equal
}

// lookupBody returns the values at the path; "*" takes every list item.
func lookupBody(v any, keys []string) []any {
	if len(keys) == 0 {
		return []any{v}
	}
	key, rest := keys[0], keys[1:]
	switch node := v.(type) {
	case map[string]any:
		if x, ok := node[key]; ok {
			return lookupBody(x, rest)
		}
	case []any:
		if key == "*" {
			var out []any
			for _, x := range node {
				out = append(out, lookupBody(x, rest)...)
			}
			return out
		}
		if i, err := strconv.Atoi(key); err == nil && i >= 0 && i < len(node) {
			return lookupBody(node[i], rest)
		}
	}
	return nil
}

// jsonEquals compares a request value (decoded with json.Number) with the
// rule's text by the value's type: a string as is, a number as a number
// ("15" = 15 = 15.0; integers exactly, however big), true/false, null, and
// an object or list as JSON.
func jsonEquals(v any, want string) bool {
	switch x := v.(type) {
	case string:
		return x == want
	case json.Number:
		return numbersEqual(string(x), strings.TrimSpace(want))
	case bool:
		return strings.TrimSpace(want) == strconv.FormatBool(x)
	case nil:
		return strings.TrimSpace(want) == "null"
	default:
		w, ok := decodeJSONNumbers([]byte(want))
		return ok && valuesEqual(v, w)
	}
}

func decodeJSONNumbers(b []byte) (any, bool) {
	if len(b) == 0 {
		return nil, false
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if dec.Decode(&v) != nil || dec.More() {
		return nil, false
	}
	return v, true
}

// numbersEqual compares two numbers as text: integers exactly, anything
// with a fraction or exponent as float64.
func numbersEqual(a, b string) bool {
	if x, ok := new(big.Int).SetString(a, 10); ok {
		if y, ok := new(big.Int).SetString(b, 10); ok {
			return x.Cmp(y) == 0
		}
	}
	x, err1 := strconv.ParseFloat(a, 64)
	y, err2 := strconv.ParseFloat(b, 64)
	return err1 == nil && err2 == nil && x == y
}

// valuesEqual compares decoded JSON, numbers by numbersEqual.
func valuesEqual(a, b any) bool {
	switch x := a.(type) {
	case json.Number:
		y, ok := b.(json.Number)
		return ok && numbersEqual(string(x), string(y))
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, xv := range x {
			yv, ok := y[k]
			if !ok || !valuesEqual(xv, yv) {
				return false
			}
		}
		return true
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !valuesEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	}
	return a == b
}
