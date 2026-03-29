package ray

import (
	"encoding/json"
	"fmt"
)

type previewMapper struct{}

func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	return payload, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}

	text := ""
	if payloads, ok := data["payloads"].([]any); ok {
		for _, p := range payloads {
			if payload, ok := p.(map[string]any); ok {
				if content, ok := payload["content"].(map[string]any); ok {
					for _, v := range content {
						text += fmt.Sprintf("%v ", v)
					}
				}
			}
		}
	}
	return text
}
