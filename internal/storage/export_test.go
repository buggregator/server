package storage

import "time"

// SetFacetBackfillPause lets tests run the backfill without pauses.
func SetFacetBackfillPause(d time.Duration) (restore func()) {
	old := facetBackfillPause
	facetBackfillPause = d
	return func() { facetBackfillPause = old }
}
