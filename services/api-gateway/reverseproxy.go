package main

import (
	"log"
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
	return httputil.NewSingleHostReverseProxy(u)
}
