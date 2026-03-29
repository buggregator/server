package httpdumps

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

	preview := map[string]any{
		"method": data["method"],
		"uri":    data["uri"],
		"host":   data["host"],
	}
	b, _ := json.Marshal(preview)
	return b, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}
	return fmt.Sprintf("%s %s %s", data["method"], data["uri"], data["host"])
}
