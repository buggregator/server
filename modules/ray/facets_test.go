package ray

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
		{"two payloads one host", `{"uuid":"u1","payloads":[{"type":"log","content":{"values":["a"]},"origin":{"function_name":"f","file":"/app/a.php","line_number":3,"hostname":"dev-laptop"}},{"type":"color","content":{"color":"red"},"origin":{"hostname":"dev-laptop"}}],"meta":{"php_version":"8.3.0"}}`,
			map[string][]string{"host": {"dev-laptop"}}},
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
