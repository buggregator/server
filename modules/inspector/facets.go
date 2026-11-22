package inspector

import (
	"encoding/json"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: the machine (host.hostname) and, when the SDK sets it, the user of
// each Inspector item.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var items []struct {
		Host struct {
			Hostname string `json:"hostname"`
		} `json:"host"`
		User json.RawMessage `json:"user"`
	}
	if json.Unmarshal(payload, &items) != nil {
		return nil
	}
	var s event.FacetSet
	for _, it := range items {
		s.Add(event.FacetHost, it.Host.Hostname)
		user := event.Object(it.User)
		for _, k := range []string{"email", "name", "id"} {
			if v := event.ScalarString(user[k]); v != "" {
				s.Add(event.FacetUser, v)
				break
			}
		}
	}
	return s.List()
}
