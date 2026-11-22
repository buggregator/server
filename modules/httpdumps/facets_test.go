package httpdumps

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

func TestFacets(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    map[string][]string
	}{
		{"host with port", `{"host":"api.local:8000","request":{"method":"GET","uri":"/"}}`, map[string][]string{"host": {"api.local"}}},
		{"plain host", `{"host":"example.test"}`, map[string][]string{"host": {"example.test"}}},
		{"sentry-trace from an outgoing call", `{"host":"api.local","request":{"headers":{"Sentry-Trace":["771a43a4192642f0b136d5159a501700-1e57b752bc6e4544-1"],"Baggage":["sentry-trace_id=771a43a4192642f0b136d5159a501700"]}}}`,
			map[string][]string{"host": {"api.local"}, "trace": {"771a43a4192642f0b136d5159a501700"}}},
		{"traceparent", `{"host":"api.local","request":{"headers":{"Traceparent":["00-4BF92F3577B34DA6A3CE929D0E0E4736-00f067aa0ba902b7-01"]}}}`,
			map[string][]string{"host": {"api.local"}, "trace": {"4bf92f3577b34da6a3ce929d0e0e4736"}}},
		{"not json", `x`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := event.FacetMap((&previewMapper{}).Facets(json.RawMessage(tc.payload)))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
