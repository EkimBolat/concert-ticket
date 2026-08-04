package main

import (
	"database/sql"
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

// dependencies bundles everything the saga needs to talk to the outside
// world, so handlers/saga.go don't reach for globals.
type dependencies struct {
	db             *sql.DB
	seatLockingURL string
	paymentURL     string
	events         *eventPublisher
}

func main() {
	port := getenv("PORT", "8083")

	db := newDB()
	defer db.Close()

	events := newEventPublisher(getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"))

	deps := &dependencies{
		db:             db,
		seatLockingURL: getenv("SEAT_LOCKING_URL", "http://localhost:8082"),
		paymentURL:     getenv("PAYMENT_URL", "http://localhost:8084"),
		events:         events,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"service": "order", "status": "ok"})
	})
	mux.HandleFunc("/orders", placeOrderHandler(deps))

	log.Printf("order service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
