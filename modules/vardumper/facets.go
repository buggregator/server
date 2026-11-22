package vardumper

import (
	"encoding/json"
	"net/url"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: a dump carries no machine name; the host of the request it was
// made in (context.request.uri) is the closest. CLI dumps have none.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Context struct {
			Request struct {
				URI string `json:"uri"`
			} `json:"request"`
		} `json:"context"`
	}
	if json.Unmarshal(payload, &data) != nil || data.Context.Request.URI == "" {
		return nil
	}
	u, err := url.Parse(data.Context.Request.URI)
	if err != nil {
		return nil
	}
	var s event.FacetSet
	s.Add(event.FacetHost, u.Hostname())
	return s.List()
}
