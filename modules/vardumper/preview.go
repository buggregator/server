package vardumper

import (
	"encoding/json"
	"fmt"
)

type previewMapper struct{}

func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var data map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, nil
	}

	// Extract the inner payload for preview.
	inner, ok := data["payload"].(map[string]any)
	if !ok {
		return payload, nil
	}

	preview := map[string]any{
		"type":  inner["type"],
		"label": inner["label"],
	}

	// For primitives, include the value in preview.
	if t, ok := inner["type"].(string); ok {
		switch t {
		case "string", "boolean", "integer", "double", "code":
			preview["value"] = inner["value"]
		default:
			// For complex types (HTML), truncate or omit in preview.
			preview["value"] = fmt.Sprintf("[%s]", t)
		}
	}

	b, _ := json.Marshal(preview)
	return b, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}

	inner, ok := data["payload"].(map[string]any)
	if !ok {
		return ""
	}

	text := ""
	if label, ok := inner["label"].(string); ok {
		text += label + " "
	}
	if t, ok := inner["type"].(string); ok {
		text += t + " "
	}
	if v, ok := inner["value"].(string); ok && len(v) < 200 {
		text += v
	}
	return text
}
