package smtp

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

func TestFacets_TraceFromHeaders(t *testing.T) {
	const id = "771a43a4192642f0b136d5159a501700"
	for _, tc := range []struct {
		name, headers string
		want          map[string][]string
	}{
		{"X-Trace-Id", "X-Trace-Id: " + id + "\r\n", map[string][]string{"trace": {id}}},
		{"sentry-trace", "Sentry-Trace: " + id + "-1e57b752bc6e4544-1\r\n", map[string][]string{"trace": {id}}},
		{"no header", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte("From: app@example.com\r\nTo: user@example.com\r\nSubject: Welcome\r\n" + tc.headers + "Content-Type: text/plain\r\n\r\nHi")
			parsed, _, err := parseEmail(raw, []string{"user@example.com"})
			if err != nil {
				t.Fatal(err)
			}
			payload, _ := json.Marshal(parsed)
			got := event.FacetMap((&previewMapper{}).Facets(payload))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
