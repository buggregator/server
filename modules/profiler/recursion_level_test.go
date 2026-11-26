package profiler

import "testing"

func TestRecursionLevel(t *testing.T) {
	for name, want := range map[string]struct {
		base  string
		level int
	}{
		"foo":                 {"foo", 0},
		"foo@2":               {"foo", 2},
		"Foo::bar@12":         {"Foo::bar", 12},
		"user@example":        {"user@example", 0},
		"foo@":                {"foo@", 0},
		"@3":                  {"@3", 0},
		"foo@-1":              {"foo@-1", 0},
		"foo@+1":              {"foo@+1", 0},
		"{closure:a.php:3}@1": {"{closure:a.php:3}", 1},
	} {
		base, level := recursionLevel(name)
		if base != want.base || level != want.level {
			t.Errorf("%q → %q %d, want %q %d", name, base, level, want.base, want.level)
		}
	}
}
