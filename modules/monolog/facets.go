package monolog

import (
	"encoding/json"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: channel, level and the machine name a HostnameProcessor puts into
// extra.hostname (Monolog sends no host by default).
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Channel   string          `json:"channel"`
		LevelName string          `json:"level_name"`
		Extra     json.RawMessage `json:"extra"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return nil
	}
	var s event.FacetSet
	s.Add(event.FacetChannel, data.Channel)
	s.Add(event.FacetLevel, data.LevelName)
	s.Add(event.FacetHost, event.ScalarString(event.Object(data.Extra)["hostname"]))
	return s.List()
}
