package main

import (
	"crypto/subtle"
	"net/http"
)

// hasInternalSecret reports whether the request carries the shared
// secret used by trusted internal callers (currently: the Order
// Service, confirming or releasing a seat as part of the purchase
// saga). release and confirm are meant to be called server-to-server
// only -- without this check, any client that can reach this service
// (through the gateway's blanket /seats/ passthrough, or directly if
// the service has its own public URL) could mark a locked seat SOLD,
// or release someone else's in-progress checkout, without ever going
// through the order/payment flow.
func hasInternalSecret(r *http.Request, secret string) bool {
	return subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Internal-Secret")), []byte(secret)) == 1
}
