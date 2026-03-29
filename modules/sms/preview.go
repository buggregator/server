package sms

import (
	"encoding/json"
	"fmt"
)

type previewMapper struct{}

// ToPreview returns full payload (same as PHP — preview includes all fields).
func (p *previewMapper) ToPreview(payload json.RawMessage) (json.RawMessage, error) {
	return payload, nil
}

func (p *previewMapper) ToSearchableText(payload json.RawMessage) string {
	var msg SmsMessage
	if json.Unmarshal(payload, &msg) != nil {
		return ""
	}
	return fmt.Sprintf("%s %s %s", msg.From, msg.To, msg.Message)
}
