package smtp

import (
	"bufio"
	"encoding/json"
	"net/textproto"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: the trace id of the request that sent the mail, from its
// sentry-trace, traceparent or X-Trace-Id header (the app adds one; mailers
// send none by default). It puts the mail on that request's timeline.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Raw string `json:"raw"`
	}
	if json.Unmarshal(payload, &data) != nil || data.Raw == "" {
		return nil
	}
	hdr, _ := textproto.NewReader(bufio.NewReader(strings.NewReader(data.Raw))).ReadMIMEHeader()
	var s event.FacetSet
	s.Add(event.FacetTrace, event.TraceIDFromHeaders(hdr.Get))
	return s.List()
}
