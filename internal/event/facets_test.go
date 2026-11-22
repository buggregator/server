package event

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFacetSet(t *testing.T) {
	var s FacetSet
	s.Add(FacetHost, " web-1 ")
	s.Add(FacetHost, "web-1") // duplicate
	s.Add(FacetEnv, "")
	s.Add(FacetUser, strings.Repeat("я", 150)) // 300 bytes, cut on a rune
	for i := 0; i < MaxFacetTags+5; i++ {
		s.AddTag(fmt.Sprintf("t%d", i), "v")
	}
	s.AddTag("", "v")

	m := FacetMap(s.List())
	if got := m[FacetHost]; len(got) != 1 || got[0] != "web-1" {
		t.Errorf("host = %v", got)
	}
	if _, ok := m[FacetEnv]; ok {
		t.Error("empty env kept")
	}
	u := m[FacetUser][0]
	if len(u) > MaxFacetValueLen || !utf8.ValidString(u) {
		t.Errorf("user not cut on a rune: len %d", len(u))
	}
	tags := 0
	for k := range m {
		if strings.HasPrefix(k, FacetTagPrefix) {
			tags++
		}
	}
	if tags != MaxFacetTags {
		t.Errorf("tags = %d, want %d", tags, MaxFacetTags)
	}
}

func TestIsFacetKey(t *testing.T) {
	for key, want := range map[string]bool{
		"host": true, "env": true, "release": true, "tag:x": true,
		"tag:": false, "type": false, "": false,
	} {
		if IsFacetKey(key) != want {
			t.Errorf("IsFacetKey(%q) = %v", key, !want)
		}
	}
}
