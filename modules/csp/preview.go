package csp

import (
	"encoding/json"
	"strings"
)

type previewMapper struct{}

// ToPreview is the payload itself: reports are small.
func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	return payload, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var pl Payload
	if json.Unmarshal(payload, &pl) != nil {
		return ""
	}
	var b strings.Builder
	for _, r := range pl.Reports {
		for _, s := range []string{r.Type, r.URL, r.Directive, r.Blocked, r.SourceFile, r.Message} {
			if s != "" {
				b.WriteString(s)
				b.WriteByte(' ')
			}
		}
	}
	return strings.TrimSpace(b.String())
}
