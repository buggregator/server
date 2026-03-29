package monolog

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
	for _, key := range []string{"message", "channel", "level", "level_name", "datetime"} {
		if v, ok := data[key]; ok {
			preview[key] = v
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
	return fmt.Sprintf("%v %v %v", data["channel"], data["level_name"], data["message"])
}
