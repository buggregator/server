// Package sameorigin tells a request from this server's own UI (or from no
// page at all) from a request another web page made, and a request from this
// machine from one that came over the network.
package sameorigin

import (
	"net"
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

// Local reports whether a request comes from this machine: the peer is a
// loopback address, the Host names this machine (a DNS-rebound page has a
// foreign Host) and no forwarding header is set. A proxy on this host that
// adds none (plain nginx proxy_pass) still looks local and cannot be told
// apart, so a proxy needs auth. In Docker the peer is the bridge gateway,
// never loopback.
func Local(r *http.Request) bool {
	for _, h := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Real-IP"} {
		if r.Header.Get(h) != "" {
			return false
		}
	}
	return loopback(r.RemoteAddr) && loopback(r.Host)
}

// loopback: "127.0.0.1:8000", "[::1]:8000", "localhost" and the like.
func loopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
