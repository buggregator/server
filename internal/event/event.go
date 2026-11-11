package event

import (
	"encoding/json"
	"errors"
	"net/http"
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

	// Merge, when set, folds this payload into a stored event with the same
	// UUID and type instead of storing a new one (Ray sends follow-ups such
	// as trace or color under the uuid of the first payload). It gets the
	// stored payload and returns the merged one.
	Merge func(stored json.RawMessage) (json.RawMessage, error)

	// Respond, when set, writes the HTTP answer after the event is stored,
	// instead of the default acknowledgement (a provider API the client
	// expects to answer in its own shape, e.g. Twilio's 201 with a sid).
	Respond func(w http.ResponseWriter)
}

// ErrNotFound: the event is not stored (e.g. deleted meanwhile).
var ErrNotFound = errors.New("event not found")

// ErrMergeLimit: a Merge refuses a follow-up because the stored event is
// full; the follow-up is dropped.
var ErrMergeLimit = errors.New("event is full")

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
