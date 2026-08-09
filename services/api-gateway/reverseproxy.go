package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// newProxy builds a reverse proxy to a downstream service. All of our
// services already use the same route paths the gateway exposes
// (/queue/..., /seats/..., /orders), so this is a pure passthrough --
// no path rewriting needed.
func newProxy(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid upstream url %q: %v", target, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(u)

	// Strip any CORS headers the upstream response might carry (from
	// seat-locking's own corsMiddleware, or headers the hosting platform
	// injects) before they get merged onto the gateway's response.
	// ReverseProxy appends upstream headers rather than replacing them,
	// so without this, "Access-Control-Allow-Origin" ends up with
	// multiple values and the browser rejects the whole response.
	// corsMiddleware (in cors.go) is the single source of truth.
	proxy.ModifyResponse = func(res *http.Response) error {
		res.Header.Del("Access-Control-Allow-Origin")
		res.Header.Del("Access-Control-Allow-Methods")
		res.Header.Del("Access-Control-Allow-Headers")
		return nil
	}

	return proxy
}
