package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// TODO (next steps for this service):
// - Redis sorted-set queue (score = arrival timestamp)\n// - Background admitter goroutine (admit N users every T seconds)\n// - Issue short-lived JWT admission tokens\n// - WebSocket/polling endpoint for live queue position

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "waiting-room",
			"status":  "ok",
		})
	})

	log.Printf("waiting-room service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
