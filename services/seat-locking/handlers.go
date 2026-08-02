package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
)

type lockRequest struct {
	UserID string `json:"userId"`
}

// path shape: /seats/{eventId}/{seatId}/lock|release|confirm
func parseSeatPath(path string) (eventID, seatID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "seats" {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

func seatsHandler(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, seatID, action, ok := parseSeatPath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body lockRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		switch action {
		case "lock":
			acquired, err := lockSeat(ctx, rdb, eventID, seatID, body.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"locked": acquired})

		case "release":
			released, err := releaseSeat(ctx, rdb, eventID, seatID, body.UserID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"released": released})

		case "confirm":
			if err := confirmSeat(ctx, rdb, eventID, seatID, body.UserID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{"confirmed": true})

		default:
			http.NotFound(w, r)
		}
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
