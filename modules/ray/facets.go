package ray

import (
	"encoding/json"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: the machine of each payload's origin (spatie/ray sends hostname).
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Payloads []struct {
			Origin struct {
				Hostname string `json:"hostname"`
			} `json:"origin"`
		} `json:"payloads"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return nil
	}
	var s event.FacetSet
	for _, pl := range data.Payloads {
		s.Add(event.FacetHost, pl.Origin.Hostname)
	}
	return s.List()
}
