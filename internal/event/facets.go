package event

import (
	"encoding/json"
	"sort"
	"strings"
	"unicode/utf8"
)

// Facet keys a feed can be filtered by. Tags use FacetTagPrefix + tag name.
const (
	FacetHost    = "host"
	FacetEnv     = "env"
	FacetUser    = "user"
	FacetChannel = "channel"
	FacetLevel   = "level"
	FacetRelease = "release"

	FacetTagPrefix = "tag:"
)

// Facet limits: long or numerous values (request ids as tags) would bloat
// the index without helping a filter.
const (
	MaxFacetValueLen = 200
	MaxFacetTags     = 50
)

// Facet is one filterable value of an event, e.g. {host, web-1}.
type Facet struct {
	Key   string
	Value string
}

// FacetExtractor is implemented by a PreviewMapper whose payloads carry
// filterable values (machine, environment, user, channel, tags).
type FacetExtractor interface {
	Facets(payload json.RawMessage) []Facet
}

// FacetFilter selects events by one facet key: an event matches when it has
// any of Values for Key (OR within a key); Not inverts it. Filters on
// different keys combine with AND.
type FacetFilter struct {
	Key    string
	Values []string
	Not    bool
}

// FacetCount is one value of a facet key with the number of events having it.
type FacetCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// IsFacetKey reports whether key is a known facet key.
func IsFacetKey(key string) bool {
	switch key {
	case FacetHost, FacetEnv, FacetUser, FacetChannel, FacetLevel, FacetRelease:
		return true
	}
	return strings.HasPrefix(key, FacetTagPrefix) && len(key) > len(FacetTagPrefix)
}

// FacetSet collects facets with trimming, length cap, tag cap and
// de-duplication. The zero value is ready to use.
type FacetSet struct {
	list []Facet
	seen map[Facet]bool
	tags int
}

// Add adds key=value; empty values are skipped.
func (s *FacetSet) Add(key, value string) {
	value = strings.TrimSpace(value)
	if value == "" || key == "" {
		return
	}
	if len(value) > MaxFacetValueLen {
		value = value[:MaxFacetValueLen]
		for !utf8.ValidString(value) {
			value = value[:len(value)-1]
		}
	}
	f := Facet{Key: key, Value: value}
	if s.seen[f] {
		return
	}
	isTag := strings.HasPrefix(key, FacetTagPrefix)
	if isTag && s.tags >= MaxFacetTags {
		return
	}
	if s.seen == nil {
		s.seen = map[Facet]bool{}
	}
	s.seen[f] = true
	if isTag {
		s.tags++
	}
	s.list = append(s.list, f)
}

// AddTag adds tag name=value as key "tag:name".
func (s *FacetSet) AddTag(name, value string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if len(name) > MaxFacetValueLen {
		return
	}
	s.Add(FacetTagPrefix+name, value)
}

// AddTags adds tags given as a JSON object {"k":"v"} or a list [["k","v"]]
// (both shapes are sent by Sentry SDKs). Non-string scalars are stringified.
func (s *FacetSet) AddTags(raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var obj map[string]any
	if json.Unmarshal(raw, &obj) == nil {
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			s.AddTag(k, ScalarString(obj[k]))
		}
		return
	}
	var pairs [][]any
	if json.Unmarshal(raw, &pairs) == nil {
		for _, p := range pairs {
			if len(p) == 2 {
				if k, ok := p[0].(string); ok {
					s.AddTag(k, ScalarString(p[1]))
				}
			}
		}
	}
}

// Object decodes raw as a JSON object; anything else (PHP sends an empty
// object as [], or null) gives nil, so one odd field never loses the rest.
func Object(raw json.RawMessage) map[string]any {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// List returns the collected facets.
func (s *FacetSet) List() []Facet { return s.list }

// ScalarString renders a JSON scalar as text; objects, arrays and null give "".
func ScalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		b, _ := json.Marshal(t)
		return string(b)
	case bool:
		if t {
			return "true"
		}
		return "false"
	}
	return ""
}

// FacetMap groups facets by key for the preview: {"host": ["web-1"]}.
func FacetMap(facets []Facet) map[string][]string {
	if len(facets) == 0 {
		return nil
	}
	m := make(map[string][]string, len(facets))
	for _, f := range facets {
		m[f.Key] = append(m[f.Key], f.Value)
	}
	return m
}
