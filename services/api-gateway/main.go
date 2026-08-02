package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// TODO (next steps for this service):
// - Route requests to downstream services\n// - Add Redis-backed rate limiting per IP/user\n// - Forward admission tokens (JWT) from Waiting Room to downstream calls

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "api-gateway",
			"status":  "ok",
		})
	})

	log.Printf("api-gateway service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
