package xdebugtrace

import (
	"encoding/json"
	"strings"
)

type previewMapper struct{}

// ToPreview keeps what a trace card shows: script or URL, duration, peak
// memory, calls and whether the trace was cut.
func (previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var p Payload
	if err := json.Unmarshal(payload, &p); err != nil || p.Summary == nil {
		return payload, nil
	}
	s := p.Summary
	b, _ := json.Marshal(map[string]any{
		"name":        p.Name,
		"url":         p.URL,
		"main":        s.Main,
		"duration":    s.Duration,
		"peak_memory": s.PeakMemory,
		"calls":       s.Calls,
		"max_depth":   s.MaxDepth,
		"truncated":   s.Truncated,
		"size":        p.Size,
	})
	return b, nil
}

func (previewMapper) ToSearchableText(payload json.RawMessage) string {
	var p Payload
	if json.Unmarshal(payload, &p) != nil {
		return ""
	}
	parts := []string{p.Name, p.URL}
	if p.Summary != nil {
		parts = append(parts, p.Summary.Main)
		for _, f := range p.Summary.Top {
			parts = append(parts, f.Func)
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
