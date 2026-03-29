package event

import (
	"encoding/json"
	"time"
)

// Event represents a stored debugging event.
type Event struct {
	UUID      string          `json:"uuid"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp float64         `json:"timestamp"`
	Project   string          `json:"project,omitempty"`
	IsPinned  bool            `json:"is_pinned"`
}

// Incoming is what an ingestion handler produces before storage.
type Incoming struct {
	UUID    string
	Type    string
	Payload json.RawMessage
	Project string
}

// Preview is the compact representation sent to the UI list and WebSocket.
type Preview struct {
	UUID           string          `json:"uuid"`
	Type           string          `json:"type"`
	Payload        json.RawMessage `json:"payload"`
	Timestamp      float64         `json:"timestamp"`
	Project        string          `json:"project,omitempty"`
	SearchableText string          `json:"searchable_text"`
	IsPinned       bool            `json:"is_pinned"`
}

// NewEvent creates an Event from an Incoming with a generated timestamp.
func NewEvent(inc *Incoming) Event {
	uuid := inc.UUID
	if uuid == "" {
		uuid = GenerateUUID()
	}
	return Event{
		UUID:      uuid,
		Type:      inc.Type,
		Payload:   inc.Payload,
		Timestamp: float64(time.Now().UnixMicro()) / 1_000_000,
		Project:   inc.Project,
	}
}
