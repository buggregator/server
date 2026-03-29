package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/buggregator/go-buggregator/internal/event"
)

// InstrumentedStore wraps event.Store with Prometheus metrics.
type InstrumentedStore struct {
	inner     event.Store
	collector *Collector
}

// NewInstrumentedStore wraps a store with metrics instrumentation.
// It also seeds the events_stored_total gauge from the database.
func NewInstrumentedStore(inner event.Store, collector *Collector, db *sql.DB) *InstrumentedStore {
	s := &InstrumentedStore{inner: inner, collector: collector}
	s.seedGauge(db)
	return s
}

// seedGauge initializes the events_stored_total gauge from current DB state.
func (s *InstrumentedStore) seedGauge(db *sql.DB) {
	rows, err := db.Query(`SELECT type, COUNT(*) FROM events GROUP BY type`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var eventType string
		var count float64
		if rows.Scan(&eventType, &count) == nil {
			s.collector.EventsStoredTotal.WithLabelValues(eventType).Set(count)
		}
	}
}

func (s *InstrumentedStore) Store(ctx context.Context, ev event.Event) error {
	start := time.Now()
	err := s.inner.Store(ctx, ev)
	s.collector.StorageQueryDuration.WithLabelValues("store").Observe(time.Since(start).Seconds())
	if err == nil {
		s.collector.EventsStoredTotal.WithLabelValues(ev.Type).Inc()
	}
	return err
}

func (s *InstrumentedStore) FindByUUID(ctx context.Context, uuid string) (*event.Event, error) {
	start := time.Now()
	ev, err := s.inner.FindByUUID(ctx, uuid)
	s.collector.StorageQueryDuration.WithLabelValues("find_by_uuid").Observe(time.Since(start).Seconds())
	return ev, err
}

func (s *InstrumentedStore) FindAll(ctx context.Context, opts event.FindOptions) ([]event.Event, error) {
	start := time.Now()
	events, err := s.inner.FindAll(ctx, opts)
	s.collector.StorageQueryDuration.WithLabelValues("find_all").Observe(time.Since(start).Seconds())
	return events, err
}

func (s *InstrumentedStore) Delete(ctx context.Context, uuid string) error {
	// Look up the event type before deleting so we can decrement the gauge.
	ev, _ := s.inner.FindByUUID(ctx, uuid)

	start := time.Now()
	err := s.inner.Delete(ctx, uuid)
	s.collector.StorageQueryDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())
	if err == nil && ev != nil {
		s.collector.EventsStoredTotal.WithLabelValues(ev.Type).Dec()
	}
	return err
}

func (s *InstrumentedStore) DeleteAll(ctx context.Context, opts event.DeleteOptions) error {
	start := time.Now()
	err := s.inner.DeleteAll(ctx, opts)
	s.collector.StorageQueryDuration.WithLabelValues("delete_all").Observe(time.Since(start).Seconds())
	if err == nil {
		// Re-seed gauge after bulk delete since we don't know exactly what was removed.
		s.collector.EventsStoredTotal.Reset()
	}
	return err
}

func (s *InstrumentedStore) Pin(ctx context.Context, uuid string) error {
	start := time.Now()
	err := s.inner.Pin(ctx, uuid)
	s.collector.StorageQueryDuration.WithLabelValues("pin").Observe(time.Since(start).Seconds())
	return err
}

func (s *InstrumentedStore) Unpin(ctx context.Context, uuid string) error {
	start := time.Now()
	err := s.inner.Unpin(ctx, uuid)
	s.collector.StorageQueryDuration.WithLabelValues("unpin").Observe(time.Since(start).Seconds())
	return err
}
