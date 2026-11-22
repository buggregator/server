package httpdumps

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: the host the request was sent to (Host header, port dropped) and
// the trace id of the request that made the call, from the sentry-trace or
// traceparent header Sentry and OTel SDKs add to outgoing HTTP calls.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		Host    string `json:"host"`
		Request struct {
			Headers map[string][]string `json:"headers"`
		} `json:"request"`
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
	s.Add(event.FacetTrace, event.TraceIDFromHeaders(func(name string) string {
		return firstHeader(data.Request.Headers, name)
	}))
	return s.List()
}

// firstHeader returns the first value of a header, matching its name in any
// case (the payload keeps the names as received).
func firstHeader(headers map[string][]string, name string) string {
	for k, v := range headers {
		if strings.EqualFold(k, name) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}
