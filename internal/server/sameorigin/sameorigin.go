// Package sameorigin tells a request from this server's own UI (or from no
// page at all) from a request another web page made.
package sameorigin

import (
	"net/http"
	"net/url"
	"strings"
)

// Check reports whether a request comes from this server's own page or
// from no page at all (curl, scripts). "same-site" is refused too: another port
// of localhost (a dev server on :5173) is the same site. A browser without
// Sec-Fetch-* is judged by Origin, which must name this Host.
func Check(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		return true
	case "":
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	o, err := url.Parse(origin)
	return err == nil && o.Host != "" && strings.EqualFold(o.Host, r.Host)
}
