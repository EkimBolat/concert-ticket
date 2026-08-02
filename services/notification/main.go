package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// TODO (next steps for this service):
// - Consume order.completed / order.failed from RabbitMQ\n// - Send confirmation/failure notification (mock: log or email stub)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "notification",
			"status":  "ok",
		})
	})

	log.Printf("notification service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
