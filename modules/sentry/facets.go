package sentry

import (
	"encoding/json"

	"github.com/buggregator/server/v2/internal/event"
)

// Facets: machine (server_name), environment, user, level, logger as
// channel, release and tags of a Sentry error event.
func (p *previewMapper) Facets(payload json.RawMessage) []event.Facet {
	var data struct {
		ServerName  string          `json:"server_name"`
		Environment string          `json:"environment"`
		Level       string          `json:"level"`
		Logger      string          `json:"logger"`
		Release     any             `json:"release"`
		Tags        json.RawMessage `json:"tags"`
		User        json.RawMessage `json:"user"`
	}
	if json.Unmarshal(payload, &data) != nil {
		return nil
	}
	var s event.FacetSet
	s.Add(event.FacetHost, data.ServerName)
	s.Add(event.FacetEnv, data.Environment)
	s.Add(event.FacetLevel, data.Level)
	s.Add(event.FacetChannel, data.Logger)
	s.Add(event.FacetRelease, event.ScalarString(data.Release))
	s.Add(event.FacetUser, sentryUser(event.Object(data.User)))
	s.AddTags(data.Tags)
	return s.List()
}

// sentryUser names the user by the first of email, username, id, ip_address.
func sentryUser(u map[string]any) string {
	for _, k := range []string{"email", "username", "id", "ip_address"} {
		if v := event.ScalarString(u[k]); v != "" {
			return v
		}
	}
	return ""
}
