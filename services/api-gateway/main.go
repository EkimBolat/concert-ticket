package main

import (
	"encoding/json"
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
	port := getenv("PORT", "8080")

	waitingRoom := newProxy(getenv("WAITING_ROOM_URL", "http://localhost:8081"))
	seatLocking := newProxy(getenv("SEAT_LOCKING_URL", "http://localhost:8082"))
	order := newProxy(getenv("ORDER_URL", "http://localhost:8083"))

	limiter := newIPRateLimiter(5, 10) // 5 req/sec sustained per IP, burst of 10

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "api-gateway",
			"status":  "ok",
		})
	})

	mux.Handle("/queue/", waitingRoom)
	mux.Handle("/seats/", seatLocking)
	mux.Handle("/orders", order)

	handler := rateLimitMiddleware(limiter, mux)

	log.Printf("api-gateway service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
