package smtp

import (
	"encoding/json"
	"fmt"
)

type previewMapper struct{}

func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var data ParsedEmail
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, nil
	}

	preview := map[string]any{
		"subject": data.Subject,
		"from":    data.From,
		"to":      data.To,
	}
	b, _ := json.Marshal(preview)
	return b, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var data ParsedEmail
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}

	text := data.Subject
	for _, addr := range data.From {
		text += fmt.Sprintf(" %s %s", addr.Name, addr.Email)
	}
	for _, addr := range data.To {
		text += fmt.Sprintf(" %s %s", addr.Name, addr.Email)
	}
	return text
}
