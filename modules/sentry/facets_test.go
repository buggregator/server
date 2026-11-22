package sentry

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/buggregator/server/v2/internal/event"
)

// TestFacets_SDKFixtures: the feed filter values of real SDK error events.
func TestFacets_SDKFixtures(t *testing.T) {
	_, feed := ingestSDK(t, "node")
	got := event.FacetMap((&previewMapper{}).Facets(feed[0].Payload))
	want := map[string][]string{
		"host":        {"node-01"},
		"env":         {"staging"},
		"level":       {"error"},
		"release":     {"shop@1.2.3"},
		"user":        {"dev@example.com"},
		"tag:feature": {"checkout"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("node facets = %v, want %v", got, want)
	}

	for _, sdk := range []string{"python", "go", "browser"} {
		t.Run(sdk, func(t *testing.T) {
			_, feed := ingestSDK(t, sdk)
			m := event.FacetMap((&previewMapper{}).Facets(feed[0].Payload))
			if len(m["env"]) == 0 || len(m["level"]) == 0 {
				t.Errorf("%s facets miss env or level: %v", sdk, m)
			}
			if sdk != "browser" && len(m["host"]) == 0 {
				t.Errorf("%s facets miss host: %v", sdk, m)
			}
		})
	}
}

func TestFacets_Shapes(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    map[string][]string
	}{
		{"tags as pairs, numeric user id", `{"tags":[["browser","Chrome"],["n",3]],"user":{"id":42},"logger":"php"}`,
			map[string][]string{"tag:browser": {"Chrome"}, "tag:n": {"3"}, "user": {"42"}, "channel": {"php"}}},
		{"username before id", `{"user":{"id":"1","username":"anna"}}`, map[string][]string{"user": {"anna"}}},
		{"empty values skipped", `{"server_name":"  ","environment":""}`, nil},
		{"not json", `xx`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := event.FacetMap((&previewMapper{}).Facets(json.RawMessage(tc.payload)))
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// PHP encodes an empty array as []: a [] user must not lose the other facets.
func TestFacets_PHPEmptyArrays(t *testing.T) {
	got := event.FacetMap((&previewMapper{}).Facets(json.RawMessage(`{"server_name":"php-1","user":[],"tags":[],"release":null}`)))
	if !reflect.DeepEqual(got, map[string][]string{"host": {"php-1"}}) {
		t.Errorf("got %v", got)
	}
}
