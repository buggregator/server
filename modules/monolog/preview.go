package monolog

import (
	"encoding/json"
	"fmt"
)

// maxPreviewBlob caps context and extra in the feed preview: a record's
// small context shows inline, a big one stays out (the UI fetches the full
// event) so the preview list stays small.
const maxPreviewBlob = 2048

type previewMapper struct{}

func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload, nil
	}

	preview := map[string]json.RawMessage{}
	for _, key := range []string{"message", "channel", "level", "level_name", "datetime"} {
		if v, ok := data[key]; ok {
			preview[key] = v
		}
	}
	for _, key := range []string{"context", "extra"} {
		v, ok := data[key]
		if !ok {
			continue
		}
		if len(v) <= maxPreviewBlob {
			preview[key] = v
		} else {
			preview[key+"_truncated"] = json.RawMessage("true")
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
