package sentry

import (
	"encoding/json"
	"strings"
)

// parseEnvelopeItems parses a Sentry envelope body into individual items.
// Envelope format: header_json\n[item_header_json\npayload\n]*
func parseEnvelopeItems(body []byte) (EnvelopeHeader, []EnvelopeItem, error) {
	var envHeader EnvelopeHeader
	var items []EnvelopeItem

	data := string(body)
	lines := splitEnvelopeLines(data)

	if len(lines) == 0 {
		return envHeader, nil, nil
	}

	// First line is the envelope header.
	if err := json.Unmarshal([]byte(lines[0]), &envHeader); err != nil {
		// Non-fatal — header may be malformed but items could still be valid.
		envHeader = EnvelopeHeader{}
	}

	// Remaining lines are item_header/payload pairs.
	i := 1
	for i < len(lines) {
		headerLine := lines[i]
		i++

		var ih ItemHeader
		if err := json.Unmarshal([]byte(headerLine), &ih); err != nil {
			// Skip malformed item header.
			continue
		}

		if ih.Type == "" {
			continue
		}

		// Next line is the payload.
		var payload string
		if i < len(lines) {
			payload = lines[i]
			i++
		}

		items = append(items, EnvelopeItem{
			Type:    ih.Type,
			Header:  json.RawMessage(headerLine),
			Payload: json.RawMessage(payload),
		})
	}

	return envHeader, items, nil
}

// splitEnvelopeLines splits envelope data by newlines, preserving empty lines
// that are part of item bodies. Trims trailing empty line.
func splitEnvelopeLines(data string) []string {
	lines := strings.Split(data, "\n")

	// Trim trailing empty lines.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return lines
}
