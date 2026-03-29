package sentry

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

	preview := map[string]any{}

	if msg, ok := data["message"].(string); ok {
		preview["message"] = msg
	}
	if logger, ok := data["logger"].(string); ok {
		preview["logger"] = logger
	}
	if level, ok := data["level"].(string); ok {
		preview["level"] = level
	}
	if platform, ok := data["platform"].(string); ok {
		preview["platform"] = platform
	}

	// Extract exception info.
	if exc, ok := data["exception"].(map[string]any); ok {
		if values, ok := exc["values"].([]any); ok && len(values) > 0 {
			if first, ok := values[0].(map[string]any); ok {
				preview["exception_type"] = first["type"]
				preview["exception_value"] = first["value"]
			}
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

	text := ""
	if msg, ok := data["message"].(string); ok {
		text += msg + " "
	}
	if exc, ok := data["exception"].(map[string]any); ok {
		if values, ok := exc["values"].([]any); ok {
			for _, v := range values {
				if e, ok := v.(map[string]any); ok {
					text += fmt.Sprintf("%v: %v ", e["type"], e["value"])
				}
			}
		}
	}
	return text
}
