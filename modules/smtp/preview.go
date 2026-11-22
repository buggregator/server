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
	// Badges for the list: which other versions the message has, and its
	// invite. Omitted for an ordinary message.
	if len(data.Parts) > 0 {
		parts := make([]map[string]string, 0, len(data.Parts))
		for _, p := range data.Parts {
			parts = append(parts, map[string]string{"type": p.Type, "label": p.Label})
		}
		preview["parts"] = parts
	}
	if c := data.Calendar; c != nil {
		preview["calendar"] = map[string]any{"summary": c.Summary, "method": c.Method, "start": c.Start, "tz": c.TZ, "all_day": c.AllDay}
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
	if data.Calendar != nil {
		text += " " + data.Calendar.Summary
	}
	return text
}
