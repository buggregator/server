package monolog

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
		{"HostnameProcessor", `{"message":"Order placed","context":{},"level":200,"level_name":"INFO","channel":"orders","datetime":"2026-11-22T10:00:00+00:00","extra":{"hostname":"web-2"}}`,
			map[string][]string{"channel": {"orders"}, "level": {"info"}, "host": {"web-2"}}},
		{"no processor", `{"message":"x","level":400,"level_name":"ERROR","channel":"app","extra":[]}`,
			map[string][]string{"channel": {"app"}, "level": {"error"}}},
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
