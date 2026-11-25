package html

import (
	"encoding/json"
)

// previewHTMLMax: markup up to this size goes into the feed preview, so the
// card can show it without loading the event.
const previewHTMLMax = 32 << 10

type previewMapper struct{}

// ToPreview: title, size, the visible text (300 characters) and the markup
// when it is small; else "truncated" and the UI loads the event.
func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var pl Payload
	if err := json.Unmarshal(payload, &pl); err != nil {
		return payload, nil
	}
	out := map[string]any{"title": pl.Title, "size": pl.Size, "text": plainText(pl.HTML, 300)}
	if len(pl.HTML) <= previewHTMLMax {
		out["html"] = pl.HTML
	} else {
		out["truncated"] = true
	}
	return json.Marshal(out)
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var pl Payload
	if json.Unmarshal(payload, &pl) != nil {
		return ""
	}
	return pl.Title + " " + plainText(pl.HTML, 10000)
}

// SearchText is what full-text search (GET /api/events?q=) finds the event
// by: the title and the visible text.
func (p *previewMapper) SearchText(payload json.RawMessage) string {
	return p.ToSearchableText(payload)
}
