package sentry

import (
	"bytes"
	"encoding/json"
)

// parseEnvelopeItems parses a Sentry envelope body into individual items.
//
// Envelope format (https://develop.sentry.dev/sdk/envelopes/):
//
//	header_json\n
//	(item_header_json\n payload\n)*
//
// Item payloads may legitimately contain literal newlines when the item header
// declares a `length` (bytes); we honor that. When length is absent we fall
// back to newline-delimited payloads.
func parseEnvelopeItems(body []byte) (EnvelopeHeader, []EnvelopeItem, error) {
	var envHeader EnvelopeHeader
	var items []EnvelopeItem

	// First line: envelope header.
	nl := bytes.IndexByte(body, '\n')
	if nl < 0 {
		// Single line — try to parse as the header but no items follow.
		_ = json.Unmarshal(body, &envHeader)
		return envHeader, nil, nil
	}
	if err := json.Unmarshal(body[:nl], &envHeader); err != nil {
		envHeader = EnvelopeHeader{}
	}
	pos := nl + 1

	for pos < len(body) {
		// Skip any blank line separators between items.
		for pos < len(body) && body[pos] == '\n' {
			pos++
		}
		if pos >= len(body) {
			break
		}

		// Item header is one line.
		nl := bytes.IndexByte(body[pos:], '\n')
		var headerLine []byte
		if nl < 0 {
			headerLine = body[pos:]
			pos = len(body)
		} else {
			headerLine = body[pos : pos+nl]
			pos = pos + nl + 1
		}

		var ih ItemHeader
		if err := json.Unmarshal(headerLine, &ih); err != nil {
			continue
		}
		if ih.Type == "" {
			continue
		}

		// Item payload: prefer the explicit length when present, otherwise read
		// up to the next newline.
		var payload []byte
		if ih.Length > 0 && pos+ih.Length <= len(body) {
			payload = body[pos : pos+ih.Length]
			pos += ih.Length
			// Skip the trailing newline (envelope spec allows but doesn't require it).
			if pos < len(body) && body[pos] == '\n' {
				pos++
			}
		} else {
			nl := bytes.IndexByte(body[pos:], '\n')
			if nl < 0 {
				payload = body[pos:]
				pos = len(body)
			} else {
				payload = body[pos : pos+nl]
				pos = pos + nl + 1
			}
		}

		items = append(items, EnvelopeItem{
			Type:    ih.Type,
			Header:  append([]byte(nil), headerLine...),
			Payload: append(json.RawMessage(nil), payload...),
		})
	}

	return envHeader, items, nil
}
