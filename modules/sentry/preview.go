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

	// Match PHP Sentry EventTypeMapper::toPreview exactly.
	preview := map[string]any{
		"message":     data["message"],
		"exception":   limitExceptionFrames(data["exception"], 3),
		"level":       data["level"],
		"platform":    data["platform"],
		"environment": data["environment"],
		"server_name": data["server_name"],
		"event_id":    data["event_id"],
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
	for _, key := range []string{"message", "level", "environment", "server_name", "platform"} {
		if v, ok := data[key].(string); ok && v != "" {
			text += v + " "
		}
	}

	if exc, ok := data["exception"].(map[string]any); ok {
		if values, ok := exc["values"].([]any); ok {
			for _, v := range values {
				if e, ok := v.(map[string]any); ok {
					text += fmt.Sprintf("%v %v ", e["type"], e["value"])
				}
			}
		}
	}
	return text
}

func limitExceptionFrames(exc any, max int) any {
	excMap, ok := exc.(map[string]any)
	if !ok {
		return exc
	}

	values, ok := excMap["values"].([]any)
	if !ok {
		return exc
	}

	for i, v := range values {
		e, ok := v.(map[string]any)
		if !ok {
			continue
		}
		st, ok := e["stacktrace"].(map[string]any)
		if !ok {
			continue
		}
		frames, ok := st["frames"].([]any)
		if !ok {
			continue
		}
		if len(frames) > max {
			st["frames"] = frames[len(frames)-max:]
		}
		e["stacktrace"] = st
		values[i] = e
	}
	excMap["values"] = values
	return excMap
}
