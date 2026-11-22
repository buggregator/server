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
		{"trace_id in extra", `{"message":"x","level_name":"INFO","channel":"app","context":{"trace_id":"ignored"},"extra":{"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"}}`,
			map[string][]string{"channel": {"app"}, "level": {"info"}, "trace": {"4bf92f3577b34da6a3ce929d0e0e4736"}}},
		{"trace_id in context", `{"message":"x","level_name":"INFO","channel":"app","context":{"trace_id":"abc"},"extra":[]}`,
			map[string][]string{"channel": {"app"}, "level": {"info"}, "trace": {"abc"}}},
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

func TestSearchText(t *testing.T) {
	payload := `{"message":"Order 42 failed","channel":"orders","level_name":"ERROR","context":{"order":{"id":42,"items":["sku-1"]},"exception":"OrderFailedException"},"extra":{"hostname":"web-2"}}`
	got := (&previewMapper{}).SearchText(json.RawMessage(payload))
	want := "Order 42 failed orders exception OrderFailedException order id 42 items sku-1 hostname web-2"
	if got != want {
		t.Errorf("SearchText =\n%q, want\n%q", got, want)
	}
	if got := (&previewMapper{}).SearchText(json.RawMessage(`x`)); got != "" {
		t.Errorf("not json: %q", got)
	}
}
