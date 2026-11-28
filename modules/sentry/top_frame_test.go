package sentry

import (
	"reflect"
	"testing"
)

func TestTopAppFrame(t *testing.T) {
	cases := []struct {
		name, stack string
		want        map[string]any
	}{
		{"empty", "", nil},
		{"bad json", "{", nil},
		{"no frames", `{"frames":[]}`, nil},
		{"last in_app wins over vendor frames after it", `{"frames":[
			{"filename":"/vendor/a.php","lineno":1,"function":"run","in_app":false},
			{"filename":"/app/Jobs/SendReceipt.php","lineno":38,"function":"handle","in_app":true},
			{"filename":"/vendor/b.php","lineno":9,"function":"throw","in_app":false}]}`,
			map[string]any{"file": "/app/Jobs/SendReceipt.php", "line": 38, "function": strPtr("handle")}},
		{"no in_app: last frame", `{"frames":[
			{"filename":"a.php","lineno":1},{"filename":"b.php","lineno":2,"function":"f"}]}`,
			map[string]any{"file": "b.php", "line": 2, "function": strPtr("f")}},
		{"abs_path when no filename, no line", `{"frames":[{"abs_path":"/srv/x.py","in_app":true}]}`,
			map[string]any{"file": "/srv/x.py", "line": nil, "function": (*string)(nil)}},
		{"no file at all", `{"frames":[{"function":"f","lineno":3}]}`, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := topAppFrame(c.stack)
			if c.want == nil {
				if got != nil {
					t.Fatalf("got %v, want nil", got)
				}
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %#v, want %#v", got, c.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
