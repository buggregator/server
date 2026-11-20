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
		peerAddress = httpPeer(span)
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

// httpPeer names the host an outgoing HTTP call went to: the URL attribute
// (Sentry http.url, OTel url.full or the older http.url/url), else OTel
// server.address[:server.port], else the URL in a Sentry-style description
// ("POST http://checkout-api/orders").
func httpPeer(span RawSpan) string {
	attrs := span.Data
	for _, key := range []string{"http.url", "url.full", "url"} {
		if host := urlHost(attrs[key]); host != "" {
			return host
		}
	}
	if host := attrs["server.address"]; host != "" {
		if port := attrs["server.port"]; port != "" {
			return host + ":" + port
		}
		return host
	}
	for _, field := range strings.Fields(span.Description) {
		if host := urlHost(field); host != "" {
			return host
		}
	}
	return ""
}

func urlHost(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return ""
	}
	return u.Host
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
