package monolog

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPreview_ContextAndExtra(t *testing.T) {
	big := `{"sql":"` + strings.Repeat("x", 3000) + `"}`
	for _, tc := range []struct {
		name                     string
		payload                  string
		context, extra           string // expected raw JSON, "" = absent
		contextTrunc, extraTrunc bool
	}{
		{"small both", `{"message":"m","channel":"app","level":200,"level_name":"INFO","context":{"order":42},"extra":{"hostname":"web-1"}}`,
			`{"order":42}`, `{"hostname":"web-1"}`, false, false},
		{"PHP empty arrays", `{"message":"m","context":[],"extra":[]}`, `[]`, `[]`, false, false},
		{"big context", `{"message":"m","context":` + big + `,"extra":{"a":1}}`, "", `{"a":1}`, true, false},
		{"big extra", `{"message":"m","context":{},"extra":` + big + `}`, `{}`, "", false, true},
		{"none", `{"message":"m"}`, "", "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := (&previewMapper{}).ToPreview(json.RawMessage(tc.payload))
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatal(err)
			}
			check := func(key, want string, trunc bool) {
				if v, ok := got[key]; (want == "") == ok || (ok && string(v) != want) {
					t.Errorf("%s = %s, want %q", key, v, want)
				}
				if _, ok := got[key+"_truncated"]; ok != trunc {
					t.Errorf("%s_truncated present = %v, want %v", key, ok, trunc)
				}
			}
			check("context", tc.context, tc.contextTrunc)
			check("extra", tc.extra, tc.extraTrunc)
			if string(got["message"]) != `"m"` {
				t.Errorf("message = %s", got["message"])
			}
		})
	}
}
