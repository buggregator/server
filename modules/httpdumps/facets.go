package httpdumps

import (
	"encoding/json"
	"net"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: the host the request was sent to (Host header, port dropped).
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Host string `json:"host"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return nil
	}
	host := data.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	var s event.FacetSet
	s.Add(event.FacetHost, host)
	return s.List()
}
