package httpdumps

import (
	"encoding/json"
	"fmt"
)

type previewMapper struct{}

// ToPreview returns full payload (PHP EventTypeMapper returns unchanged).
func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	return payload, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data map[string]any
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}

	host, _ := data["host"].(string)
	req, _ := data["request"].(map[string]any)
	if req == nil {
		return host
	}

	return fmt.Sprintf("%s %s %s", host, req["method"], req["uri"])
}
