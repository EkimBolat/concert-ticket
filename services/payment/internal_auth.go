package main

import (
	"crypto/subtle"
	"net/http"
)

// requireInternalSecret rejects any request that doesn't carry the
// shared secret used by trusted internal callers (currently: the Order
// Service). /charge and /refund are meant to be called server-to-server
// only -- without this check, anyone who can reach this service
// directly (e.g. its own public URL, if it has one) could charge or
// refund an arbitrary order without ever going through the order saga.
func requireInternalSecret(secret string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Secret")), []byte(secret)) != 1 {
			http.Error(w, "forbidden: server-to-server only", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
