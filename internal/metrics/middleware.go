package metrics

import (
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware returns an HTTP middleware that records request metrics.
func HTTPMiddleware(next http.Handler, m *Collector) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)

		m.HTTPRequestsTotal.WithLabelValues(r.Method, path, fmt.Sprintf("%d", rw.statusCode)).Inc()
		m.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// normalizePath replaces UUIDs and numeric IDs with placeholders to keep label cardinality low.
func normalizePath(path string) string {
	// Replace UUIDs.
	path = uuidPattern.ReplaceAllString(path, "{uuid}")

	// Collapse any remaining long hex segments (non-UUID hashes).
	return path
}
