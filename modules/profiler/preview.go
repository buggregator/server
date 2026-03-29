package profiler

import (
	"encoding/json"
	"fmt"
	"strings"
)

type previewMapper struct{}

// ToPreview matches PHP Profiler EventTypeMapper::toPreview.
func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, nil
	}

	preview := map[string]any{
		"peaks":    data["peaks"],
		"tags":     data["tags"],
		"app_name": data["app_name"],
		"hostname": data["hostname"],
		"date":     data["date"],
	}

	b, _ := json.Marshal(preview)
	return b, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}

	var parts []string
	if v, ok := data["app_name"].(string); ok && v != "" {
		parts = append(parts, v)
	}
	if v, ok := data["hostname"].(string); ok && v != "" {
		parts = append(parts, v)
	}
	if tags, ok := data["tags"].(map[string]any); ok {
		for k, v := range tags {
			parts = append(parts, fmt.Sprintf("%s:%v", k, v))
		}
	}

	return strings.Join(parts, " ")
}
