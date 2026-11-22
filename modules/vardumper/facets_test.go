package vardumper

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
		{"request context", `{"payload":{"type":"string","value":"x"},"context":{"request":{"uri":"http://shop.local:8080/cart?x=1","method":"GET"},"source":{"name":"a.php"}}}`,
			map[string][]string{"host": {"shop.local"}}},
		{"cli context", `{"payload":{"type":"string","value":"x"},"context":{"cli":{"command_line":"bin/console app:run"}}}`, nil},
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
