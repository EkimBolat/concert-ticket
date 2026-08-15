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
	port := getenv("PORT", "8084")
	internalSecret := getenv("INTERNAL_SECRET", "dev-internal-secret-change-me")

	db := newDB()
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"service": "payment", "status": "ok"})
	})
	mux.HandleFunc("/charge", requireInternalSecret(internalSecret, chargeHandler(db)))
	mux.HandleFunc("/refund", requireInternalSecret(internalSecret, refundHandler(db)))

	log.Printf("payment service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
