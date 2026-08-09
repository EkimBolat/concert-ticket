package main

import "net/http"

// corsMiddleware lets browser-based clients (the demo page) call the
// gateway from a different origin. Wide open ("*") -- this is a demo
// project, not something handling real payments. It's the outermost
// wrapper so OPTIONS preflight requests get answered before hitting rate
// limiting or the admission-token check (which would otherwise reject a
// preflight for having no Authorization header).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
