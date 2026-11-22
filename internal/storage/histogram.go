package storage

import (
	"context"
	"database/sql"
	"math"
	"strings"

	"github.com/buggregator/server/v2/internal/event"
)

// MaxHistogramBuckets caps the buckets of one histogram: a wider window gets
// a wider bucket.
const MaxHistogramBuckets = 1440

// HistogramBucket is the number of events in one time bucket, split by level.
type HistogramBucket struct {
	// T is the bucket start, unix seconds.
	T      int64          `json:"t"`
	Count  int            `json:"count"`
	Levels map[string]int `json:"levels"`
}

// Histogram counts the events matching opts (type, project, window, filters,
// search) per bucket of bucketSec seconds, split by their level facet;
// events without a level only add to Count. Empty buckets are left out.
// It returns the bucket width used: bucketSec, or wider so the selection
// spans at most MaxHistogramBuckets buckets.
func (s *SQLiteStore) Histogram(ctx context.Context, opts event.FindOptions, bucketSec int64) ([]HistogramBucket, int64, error) {
	if bucketSec <= 0 {
		bucketSec = 60
	}
	conds, args := eventConditions(opts, "e", "")
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}

	from, to := opts.From, opts.To
	if from == 0 || to == 0 {
		var lo, hi sql.NullFloat64
		if err := s.db.QueryRowContext(ctx,
			`SELECT MIN(CAST(e.timestamp AS REAL)), MAX(CAST(e.timestamp AS REAL)) FROM events e`+where, args...,
		).Scan(&lo, &hi); err != nil {
			return nil, bucketSec, err
		}
		if !lo.Valid {
			return []HistogramBucket{}, bucketSec, nil
		}
		if from == 0 {
			from = lo.Float64
		}
		if to == 0 {
			to = hi.Float64
		}
	}
	if span := to - from; span > float64(bucketSec*MaxHistogramBuckets) {
		n := int64(math.Ceil(span / float64(bucketSec*MaxHistogramBuckets)))
		bucketSec *= n
	}

	query := `SELECT CAST(CAST(e.timestamp AS REAL) / ? AS INTEGER) * ? AS t, COALESCE(f.value, ''), COUNT(*)
		FROM events e LEFT JOIN event_facets f ON f.uuid = e.uuid AND f.key = '` + event.FacetLevel + `'` +
		where + ` GROUP BY t, f.value ORDER BY t`
	rows, err := s.db.QueryContext(ctx, query, append([]any{bucketSec, bucketSec}, args...)...)
	if err != nil {
		return nil, bucketSec, err
	}
	defer rows.Close()
	out := []HistogramBucket{}
	for rows.Next() {
		var t int64
		var level string
		var n int
		if err := rows.Scan(&t, &level, &n); err != nil {
			return nil, bucketSec, err
		}
		if len(out) == 0 || out[len(out)-1].T != t {
			out = append(out, HistogramBucket{T: t, Levels: map[string]int{}})
		}
		b := &out[len(out)-1]
		b.Count += n
		if level != "" {
			b.Levels[level] += n
		}
	}
	return out, bucketSec, rows.Err()
}
