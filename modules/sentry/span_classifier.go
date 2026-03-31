package sentry

import (
	"net/url"
	"strings"
)

// classifySpan extracts service graph metadata from raw span attributes.
// Returns peer_type ("db", "http", "cache", "queue", "unknown") and peer_address.
func classifySpan(span RawSpan) (peerType string, peerAddress string) {
	op := span.Op
	attrs := span.Data

	switch {
	case strings.HasPrefix(op, "db."):
		peerType = "db"
		peerAddress = firstNonEmpty(
			attrs["server.address"],
			attrs["db.server.address"],
			attrs["db.name"],
		)
	case op == "http.client":
		peerType = "http"
		if raw := attrs["http.url"]; raw != "" {
			if u, err := url.Parse(raw); err == nil {
				peerAddress = u.Host
			}
		}
	case strings.HasPrefix(op, "cache."):
		peerType = "cache"
		peerAddress = attrs["server.address"]
	case strings.HasPrefix(op, "queue."):
		peerType = "queue"
		peerAddress = attrs["messaging.destination"]
	default:
		peerType = "unknown"
	}

	return
}

// extractServiceName extracts the service name from span attributes or SDK info.
func extractServiceName(span RawSpan, sdk *SDK) string {
	if name := span.Data["service.name"]; name != "" {
		return name
	}
	if sdk != nil && sdk.Name != "" {
		return sdk.Name
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
