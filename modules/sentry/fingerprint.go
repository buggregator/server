package sentry

import (
	"crypto/sha256"
	"encoding/hex"
)

// computeFingerprint generates a 16 hex-char fingerprint for grouping error events.
// Uses SHA256 of "exception_type|first_100_chars_of_value" for exception events,
// or "message|first_100_chars_of_message" for message events.
func computeFingerprint(event *ErrorEvent) string {
	var src string

	if event.Exception != nil && len(event.Exception.Values) > 0 {
		e := event.Exception.Values[0]
		val := e.Value
		if len(val) > 100 {
			val = val[:100]
		}
		src = e.Type + "|" + val
	} else {
		msg := event.effectiveMessage()
		if len(msg) > 100 {
			msg = msg[:100]
		}
		src = "message|" + msg
	}

	h := sha256.Sum256([]byte(src))
	return hex.EncodeToString(h[:8]) // 16 hex chars
}
