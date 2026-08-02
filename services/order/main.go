package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// TODO (next steps for this service):
// - Saga orchestration: reserve seat -> charge payment -> confirm\n// - Compensating action: release seat lock on payment failure\n// - Publish order.completed / order.failed events to RabbitMQ\n// - Persist orders in Postgres (orderdb)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "order",
			"status":  "ok",
		})
	})

	log.Printf("order service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
