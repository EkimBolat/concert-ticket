package main

import (
	"log"
	"net/http"
	"os"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := getenv("PORT", "8082")

	rdb := newRedisClient()
	manager := newHubManager(rdb)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"service": "seat-locking", "status": "ok"})
	})

	// WebSocket: /ws/{eventId} — live seat map updates for one event.
	mux.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		eventID := r.URL.Path[len("/ws/"):]
		if eventID == "" {
			http.Error(w, "eventId is required", http.StatusBadRequest)
			return
		}
		handleWS(manager, w, r, eventID)
	})

	// REST: /seats/{eventId}/{seatId}/lock|release|confirm
	mux.HandleFunc("/seats/", seatsHandler(rdb))

	log.Printf("seat-locking service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
