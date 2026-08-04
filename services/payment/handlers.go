package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type chargeRequest struct {
	OrderID         int64  `json:"orderId"`
	UserID          string `json:"userId"`
	AmountCents     int64  `json:"amountCents"`
	SimulateFailure bool   `json:"simulateFailure"`
}

// POST /charge — idempotent by orderId: charging the same order twice
// returns the original result instead of charging again. This is the
// idempotency guarantee the whole point of this mock service demonstrates.
func chargeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req chargeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == 0 || req.UserID == "" {
			http.Error(w, "orderId and userId are required", http.StatusBadRequest)
			return
		}

		var existingStatus string
		err := db.QueryRow(`SELECT status FROM transactions WHERE order_id = $1`, req.OrderID).Scan(&existingStatus)
		if err == nil {
			writeJSON(w, map[string]any{"status": existingStatus, "idempotent": true})
			return
		}
		if err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Fake gateway logic: fails if the caller explicitly asks it to
		// (used to test the saga's compensating actions) or if the amount
		// is invalid.
		status := "succeeded"
		if req.SimulateFailure || req.AmountCents <= 0 {
			status = "failed"
		}

		if _, err := db.Exec(
			`INSERT INTO transactions (order_id, user_id, amount_cents, status) VALUES ($1, $2, $3, $4)`,
			req.OrderID, req.UserID, req.AmountCents, status,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{"status": status, "idempotent": false})
	}
}

// POST /refund — the compensating action for a charge that later needs to
// be undone (e.g. the seat couldn't be confirmed after payment succeeded).
func refundHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			OrderID int64 `json:"orderId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == 0 {
			http.Error(w, "orderId is required", http.StatusBadRequest)
			return
		}

		res, err := db.Exec(`UPDATE transactions SET status = 'refunded' WHERE order_id = $1 AND status = 'succeeded'`, req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		affected, _ := res.RowsAffected()
		writeJSON(w, map[string]any{"refunded": affected > 0})
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
