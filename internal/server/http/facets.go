package http

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
	"github.com/buggregator/server/v2/internal/storage"
)

// facetParams are the query parameters of feed filters and the facet key
// each one selects. "tag" takes name:value; any of them prefixed with "not."
// excludes instead.
var facetParams = []string{
	event.FacetHost, event.FacetEnv, event.FacetUser,
	event.FacetChannel, event.FacetLevel, event.FacetRelease, event.FacetTrace,
}

// parseFacetFilters reads feed filters from the query: repeated values of
// one parameter are OR-ed, different parameters AND-ed.
//
//	?host=web-1&host=web-2&env=staging&tag=feature:checkout&not.level=debug
func parseFacetFilters(q url.Values) []event.FacetFilter {
	byKey := map[string]*event.FacetFilter{}
	var order []string
	add := func(key, value string, not bool) {
		value = strings.TrimSpace(value)
		if value == "" || !event.IsFacetKey(key) {
			return
		}
		id := key
		if not {
			id = "not." + key
		}
		f, ok := byKey[id]
		if !ok {
			f = &event.FacetFilter{Key: key, Not: not}
			byKey[id] = f
			order = append(order, id)
		}
		f.Values = append(f.Values, value)
	}
	for _, not := range []bool{false, true} {
		prefix := ""
		if not {
			prefix = "not."
		}
		for _, key := range facetParams {
			for _, v := range q[prefix+key] {
				add(key, v, not)
			}
		}
		for _, v := range q[prefix+"tag"] {
			name, value, ok := strings.Cut(v, ":")
			if ok && strings.TrimSpace(name) != "" {
				add(event.FacetTagPrefix+strings.TrimSpace(name), value, not)
			}
		}
	}
	filters := make([]event.FacetFilter, 0, len(order))
	for _, id := range order {
		filters = append(filters, *byKey[id])
	}
	return filters
}

// facetCounter is the part of the store that counts facet values
// (storage.SQLiteStore, and metrics.InstrumentedStore around it).
type facetCounter interface {
	FacetCounts(ctx context.Context, opts event.FindOptions, perKey int) (map[string][]event.FacetCount, error)
}

// facetsHandler answers GET /api/events/facets: for each facet key, its
// values with event counts under the same type/project/window/filters as
// the feed. A key's counts ignore that key's own filter.
func facetsHandler(store event.Store, lim ListLimits) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fc, ok := store.(facetCounter)
		if !ok {
			writeJSON(w, map[string]any{"data": map[string]any{}, "keys": []string{}})
			return
		}
		opts := parseListOptions(r, lim)
		counts, err := fc.FacetCounts(r.Context(), opts, 0)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		writeJSON(w, map[string]any{"data": counts, "keys": keys})
	}
}

// histogramer is the part of the store that counts events per time bucket.
type histogramer interface {
	Histogram(ctx context.Context, opts event.FindOptions, bucketSec int64) ([]storage.HistogramBucket, int64, error)
}

// histogramHandler answers GET /api/events/histogram: events per time
// bucket (?bucket=seconds, default 60) split by level, under the same type,
// project, window, filters and search as the feed.
func histogramHandler(store event.Store, lim ListLimits) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h, ok := store.(histogramer)
		if !ok {
			writeJSON(w, map[string]any{"bucket": 60, "data": []any{}})
			return
		}
		opts := parseListOptions(r, lim)
		var bucket int64 = 60
		if v := r.URL.Query().Get("bucket"); v != "" {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n <= 0 {
				writeError(w, "bucket: seconds, a positive integer", http.StatusBadRequest)
				return
			}
			bucket = n
		}
		data, used, err := h.Histogram(r.Context(), opts, bucket)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"bucket": used, "data": data})
	}
}
