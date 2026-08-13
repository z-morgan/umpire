package server

import (
	"net"
	"net/http"
	"net/url"
)

// localhostOnly rejects requests that do not originate from the local machine
// before they reach a handler. The loopback bind (see New) stops requests from
// other hosts, but not requests the user's own browser is tricked into making
// by a page they visit. Two attacks this closes:
//
//   - DNS rebinding: an attacker-controlled domain re-resolves to 127.0.0.1
//     after its page loads, so a fetch of /api/diff looks same-origin to the
//     browser and the page can read the response. A forged host cannot name
//     localhost, so validating Host is the load-bearing half.
//   - Fire-and-forget CSRF: a foreign page POSTs to /api/shutdown or a write
//     endpoint without needing to read the reply. Rejecting a present-but-
//     foreign Origin covers the POST paths.
func localhostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostIsLocal(r.Host) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && originIsForeign(r.Header.Get("Origin")) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// hostIsLocal reports whether the Host header names the local machine. A port,
// if present, is ignored.
func hostIsLocal(host string) bool {
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}
	switch hostname {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	return false
}

// originIsForeign reports whether a present Origin points somewhere other than
// the local machine. An absent Origin is not treated as foreign: it is omitted
// on same-origin GETs and on some cross-origin simple requests, so requiring
// it would reject legitimate traffic.
func originIsForeign(origin string) bool {
	if origin == "" {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return true
	}
	return !hostIsLocal(u.Host)
}
