package main

import (
	"encoding/json"
	"net/http"
)

// POST /orders
func placeOrderHandler(deps *dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req placeOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.EventID == "" || req.SeatID == "" || req.UserID == "" || req.AmountCents <= 0 {
			http.Error(w, "eventId, seatId, userId and a positive amountCents are required", http.StatusBadRequest)
			return
		}

		result, err := placeOrder(deps, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, result)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
