package main

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

const schema = `
CREATE TABLE IF NOT EXISTS transactions (
	order_id BIGINT PRIMARY KEY,
	user_id TEXT NOT NULL,
	amount_cents BIGINT NOT NULL,
	status TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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
