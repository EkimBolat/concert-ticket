package main

import "net/http"

// corsMiddleware lets the browser-based seat map demo (services/../demo,
// served from a different origin than this API) call these endpoints
// directly. Wide open ("*") because this is a demo project handling fake
// seats, not real payments -- a production deployment would restrict this
// to the actual frontend's origin.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
