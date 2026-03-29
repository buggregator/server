package inspector

import (
	"encoding/json"
	"fmt"
)

type previewMapper struct{}

func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	return payload, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data []map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}

	text := ""
	for _, item := range data {
		if t, ok := item["type"].(string); ok {
			text += t + " "
		}
		if model, ok := item["model"].(string); ok {
			text += model + " "
		}
		if host, ok := item["host"].(map[string]any); ok {
			text += fmt.Sprintf("%v ", host["hostname"])
		}
	}
	return text
}
