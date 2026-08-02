package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// TODO (next steps for this service):
// - Mock charge/refund endpoints simulating an external gateway\n// - Idempotency: same order ID can never be charged twice\n// - Persist transactions in Postgres (paymentdb)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "payment",
			"status":  "ok",
		})
	})

	log.Printf("payment service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
