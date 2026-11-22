package monolog

import (
	"encoding/json"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: channel, level, the machine name a HostnameProcessor puts into
// extra.hostname (Monolog sends no host by default) and the trace id a
// processor in the app puts into extra.trace_id or context.trace_id.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Channel   string          `json:"channel"`
		LevelName string          `json:"level_name"`
		Extra     json.RawMessage `json:"extra"`
		Context   json.RawMessage `json:"context"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return nil
	}
	var s event.FacetSet
	s.Add(event.FacetChannel, data.Channel)
	s.Add(event.FacetLevel, data.LevelName)
	extra := event.Object(data.Extra)
	s.Add(event.FacetHost, event.ScalarString(extra["hostname"]))
	trace := event.ScalarString(extra["trace_id"])
	if trace == "" {
		trace = event.ScalarString(event.Object(data.Context)["trace_id"])
	}
	s.Add(event.FacetTrace, trace)
	return s.List()
}

// SearchText is what full-text search finds a log line by: the message, the
// channel and the keys and values of context and extra.
func (p *previewMapper) SearchText(payload json.RawMessage) string {
	var data struct {
		Message string          `json:"message"`
		Channel string          `json:"channel"`
		Context json.RawMessage `json:"context"`
		Extra   json.RawMessage `json:"extra"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return ""
	}
	return strings.Join([]string{data.Message, data.Channel, event.FlattenJSON(data.Context), event.FlattenJSON(data.Extra)}, " ")
}
