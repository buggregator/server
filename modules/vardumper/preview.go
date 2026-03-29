package vardumper

import (
	"encoding/json"
)

// previewMapper for VarDumper returns the full payload as-is.
// This matches the original PHP Buggregator where VarDumper has no
// EventTypeMapper — the complete payload (with HTML) is broadcast.
type previewMapper struct{}

func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	// Return full payload — frontend needs payload.payload.value with complete HTML/value.
	return payload, nil
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
