package profiler

import (
	"encoding/json"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: the machine (hostname) and tags of a profile.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Hostname string          `json:"hostname"`
		Tags     json.RawMessage `json:"tags"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return nil
	}
	var s event.FacetSet
	s.Add(event.FacetHost, data.Hostname)
	s.AddTags(data.Tags)
	return s.List()
}
