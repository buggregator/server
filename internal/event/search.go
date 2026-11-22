package event

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxSearchTextLen caps the text indexed for search per event: the index
// takes a few times the size of its text.
const MaxSearchTextLen = 4096

// SearchTextExtractor is implemented by a PreviewMapper whose events can be
// found by full-text search (GET /api/events?q=).
type SearchTextExtractor interface {
	SearchText(payload json.RawMessage) string
}

// SearchText returns the text to index for a payload of eventType, or ""
// when its mapper indexes none.
func (r *PreviewRegistry) SearchText(eventType string, payload json.RawMessage) string {
	if sx, ok := r.Get(eventType).(SearchTextExtractor); ok {
		return TruncateText(sx.SearchText(payload), MaxSearchTextLen)
	}
	return ""
}

// FlattenJSON writes the keys and scalar values of a JSON document as
// space-separated text ("user_id 42 email a@b.c"), keys in sorted order.
func FlattenJSON(raw json.RawMessage) string {
	var v any
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil {
		return ""
	}
	var b strings.Builder
	flatten(&b, v)
	return strings.TrimSpace(b.String())
}

func flatten(b *strings.Builder, v any) {
	if b.Len() > MaxSearchTextLen {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(k)
			b.WriteByte(' ')
			flatten(b, t[k])
		}
	case []any:
		for _, item := range t {
			flatten(b, item)
		}
	default:
		if s := ScalarString(t); s != "" {
			b.WriteString(s)
			b.WriteByte(' ')
		}
	}
}

// TruncateText cuts s to at most n bytes on a rune boundary.
func TruncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
