package inspector

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
		{"transaction with host and user", `[{"model":"transaction","type":"request","name":"GET /","host":{"hostname":"api-1","ip":"10.0.0.2","os":"Linux"},"user":{"id":5,"name":"Anna"}}]`,
			map[string][]string{"host": {"api-1"}, "user": {"Anna"}}},
		{"no user", `[{"model":"segment","host":{"hostname":"api-1"}}]`, map[string][]string{"host": {"api-1"}}},
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
