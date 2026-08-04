package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

const schema = `
CREATE TABLE IF NOT EXISTS orders (
	id BIGSERIAL PRIMARY KEY,
	event_id TEXT NOT NULL,
	seat_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	amount_cents BIGINT NOT NULL,
	status TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func newDB() *sql.DB {
	dsn := getenv("DATABASE_URL", "postgres://gatekeeper:gatekeeper@localhost:5432/gatekeeper?sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("failed to run schema migration: %v", err)
	}
	return db
}

func insertOrder(db *sql.DB, eventID, seatID, userID string, amountCents int64, status string) (int64, error) {
	var id int64
	err := db.QueryRow(
		`INSERT INTO orders (event_id, seat_id, user_id, amount_cents, status) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		eventID, seatID, userID, amountCents, status,
	).Scan(&id)
	return id, err
}

func updateOrderStatus(db *sql.DB, id int64, status string) error {
	_, err := db.Exec(`UPDATE orders SET status = $1, updated_at = now() WHERE id = $2`, status, id)
	return err
}
