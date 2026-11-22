package event

import "strings"

// TraceIDFromHeaders finds the trace id of the request that sent a message
// (a mail, an HTTP call) in its headers. get returns a header value by its
// canonical name. It reads, in order:
//   - sentry-trace: "<trace id>-<span id>[-<sampled>]", which Sentry SDKs
//     add to outgoing HTTP calls;
//   - traceparent (W3C): "00-<trace id>-<span id>-<flags>", from OTel and
//     current Sentry SDKs;
//   - X-Trace-Id: the trace id alone, for apps that set it on mails.
//
// The id is returned in lower case, or "" when no header holds a valid one.
func TraceIDFromHeaders(get func(name string) string) string {
	if v := strings.TrimSpace(get("Sentry-Trace")); v != "" {
		if id, _, _ := strings.Cut(v, "-"); isTraceID(id) {
			return strings.ToLower(id)
		}
	}
	if v := strings.TrimSpace(get("Traceparent")); v != "" {
		if parts := strings.Split(v, "-"); len(parts) >= 3 && isTraceID(parts[1]) {
			return strings.ToLower(parts[1])
		}
	}
	if v := strings.TrimSpace(get("X-Trace-Id")); isTraceID(v) {
		return strings.ToLower(v)
	}
	return ""
}

// isTraceID reports whether s is 32 hex digits and not all zero.
func isTraceID(s string) bool {
	if len(s) != 32 {
		return false
	}
	zero := true
	for _, c := range s {
		switch {
		case c == '0':
		case c >= '1' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
			zero = false
		default:
			return false
		}
	}
	return !zero
}
