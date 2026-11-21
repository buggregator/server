package broadcast

import (
	"encoding/json"
	"strings"
)

// previewLimit bounds the data kept in the preview; the event has it all.
const previewLimit = 2048

type previewMapper struct{}

// ToPreview is the event with its data cut to previewLimit bytes.
func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var e Event
	if json.Unmarshal(payload, &e) != nil {
		return payload, nil
	}
	if len(e.Data) > previewLimit {
		e.Data = e.Data[:previewLimit]
		e.DataJSON = nil
	}
	return json.Marshal(e)
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var e Event
	if json.Unmarshal(payload, &e) != nil {
		return ""
	}
	parts := append([]string{e.Name, e.AppID}, e.Channels...)
	if !e.Encrypted {
		d := e.Data
		if len(d) > previewLimit {
			d = d[:previewLimit]
		}
		parts = append(parts, d)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
