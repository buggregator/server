package sentry

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
)

// Minimal source-map (v3) support, enough to turn a transformed/bundled dev
// frame back into the developer's original source. Dev servers (Vite, webpack)
// serve compiled modules with an inline `//# sourceMappingURL=data:...` map whose
// `sourcesContent` carries the original file; we decode the VLQ `mappings` to
// translate the frame's generated (line, col) into the original position.

type sourceMap struct {
	Version        int      `json:"version"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
	Mappings       string   `json:"mappings"`
	Names          []string `json:"names"`

	origLineCache map[int][]string // sourceIndex -> split sourcesContent
}

const b64alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var b64lookup = func() [256]int8 {
	var t [256]int8
	for i := range t {
		t[i] = -1
	}
	for i := 0; i < len(b64alphabet); i++ {
		t[b64alphabet[i]] = int8(i)
	}
	return t
}()

// decodeVLQ decodes a base64 VLQ-encoded segment into its integer fields.
func decodeVLQ(seg string) []int {
	var out []int
	shift, value := 0, 0
	for i := 0; i < len(seg); i++ {
		digit := b64lookup[seg[i]]
		if digit < 0 {
			return out
		}
		d := int(digit)
		cont := d&32 != 0
		d &= 31
		value += d << shift
		shift += 5
		if !cont {
			neg := value&1 != 0
			value >>= 1
			if neg {
				value = -value
			}
			out = append(out, value)
			shift, value = 0, 0
		}
	}
	return out
}

// extractSourceMap finds and parses the source map referenced by a transformed
// source file. fileURL is the URL the text was fetched from (used to resolve a
// relative external map). fetch retrieves an external .map when needed.
func extractSourceMap(text, fileURL string, fetch sourceFetcher) (*sourceMap, bool) {
	marker := "sourceMappingURL="
	idx := strings.LastIndex(text, marker)
	if idx < 0 {
		return nil, false
	}
	ref := text[idx+len(marker):]
	if nl := strings.IndexAny(ref, "\r\n"); nl >= 0 {
		ref = ref[:nl]
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, false
	}

	var raw []byte
	if strings.HasPrefix(ref, "data:") {
		comma := strings.IndexByte(ref, ',')
		if comma < 0 {
			return nil, false
		}
		meta, payload := ref[len("data:"):comma], ref[comma+1:]
		if strings.Contains(meta, "base64") {
			dec, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				return nil, false
			}
			raw = dec
		} else {
			dec, err := url.QueryUnescape(payload)
			if err != nil {
				return nil, false
			}
			raw = []byte(dec)
		}
	} else {
		mapURL := resolveURL(fileURL, ref)
		body, ok := fetch(mapURL)
		if !ok {
			return nil, false
		}
		raw = []byte(body)
	}

	var sm sourceMap
	if err := json.Unmarshal(raw, &sm); err != nil {
		return nil, false
	}
	if sm.Mappings == "" || len(sm.SourcesContent) == 0 {
		return nil, false
	}
	return &sm, true
}

func resolveURL(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
}

// originalPosition translates a generated 1-based line and 0-based column into
// the original source: index into Sources/SourcesContent and 0-based line.
func (sm *sourceMap) originalPosition(genLine, genCol int) (srcIndex, origLine, origCol int, ok bool) {
	lines := strings.Split(sm.Mappings, ";")
	if genLine < 1 || genLine > len(lines) {
		return 0, 0, 0, false
	}

	// Source index / original line / original column are delta-encoded across the
	// whole mappings string; the generated column resets at each generated line.
	curSrc, curLine, curCol := 0, 0, 0
	found := false
	var fSrc, fLine, fCol int
	var firstSrc, firstLine, firstCol int
	haveFirst := false

	for li := 0; li < genLine; li++ {
		genColAbs := 0
		for _, seg := range strings.Split(lines[li], ",") {
			if seg == "" {
				continue
			}
			f := decodeVLQ(seg)
			if len(f) == 0 {
				continue
			}
			genColAbs += f[0]
			if len(f) >= 4 {
				curSrc += f[1]
				curLine += f[2]
				curCol += f[3]
			}
			if li == genLine-1 && len(f) >= 4 {
				if !haveFirst {
					firstSrc, firstLine, firstCol = curSrc, curLine, curCol
					haveFirst = true
				}
				if genColAbs <= genCol {
					fSrc, fLine, fCol = curSrc, curLine, curCol
					found = true
				}
			}
		}
	}

	if found {
		return fSrc, fLine, fCol, true
	}
	if haveFirst {
		return firstSrc, firstLine, firstCol, true
	}
	return 0, 0, 0, false
}

// originalSourceLines returns the split content of the given source index.
func (sm *sourceMap) originalSourceLines(srcIndex int) ([]string, bool) {
	if srcIndex < 0 || srcIndex >= len(sm.SourcesContent) {
		return nil, false
	}
	content := sm.SourcesContent[srcIndex]
	if content == "" {
		return nil, false
	}
	if sm.origLineCache == nil {
		sm.origLineCache = map[int][]string{}
	}
	if cached, ok := sm.origLineCache[srcIndex]; ok {
		return cached, true
	}
	lines := strings.Split(content, "\n")
	sm.origLineCache[srcIndex] = lines
	return lines, true
}

// sourceName returns a display name for a resolved source index.
func (sm *sourceMap) sourceName(srcIndex int) string {
	if srcIndex < 0 || srcIndex >= len(sm.Sources) {
		return ""
	}
	return sm.Sources[srcIndex]
}
