package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type joinRequest struct {
	UserID string `json:"userId"`
}

// POST /queue/{eventId}/join
func joinHandler(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, ok := pathEventID(r.URL.Path, "/queue/", "/join")
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body joinRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		key := queueKey(eventID)

		// NX add: only sets a score if this member isn't already queued,
		// so refreshing the page doesn't send you to the back of the line.
		added, err := rdb.ZAddNX(ctx, key, redis.Z{
			Score:  float64(time.Now().UnixNano()),
			Member: body.UserID,
		}).Result()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if added > 0 {
			rdb.SAdd(ctx, activeQueuesKey, eventID)
		}

		rank, err := rdb.ZRank(ctx, key, body.UserID).Result()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{
			"status":   "waiting",
			"position": rank + 1,
		})
	}
}

// GET /queue/{eventId}/status?userId=...
func statusHandler(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		eventID, ok := pathEventID(r.URL.Path, "/queue/", "/status")
		if !ok {
			http.NotFound(w, r)
			return
		}
		userID := r.URL.Query().Get("userId")
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		ctx := context.Background()

		// Already admitted? Hand back the token.
		token, err := rdb.Get(ctx, admittedKey(eventID, userID)).Result()
		if err == nil {
			writeJSON(w, map[string]any{"status": "admitted", "token": token})
			return
		}
		if err != redis.Nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Still in line?
		rank, err := rdb.ZRank(ctx, queueKey(eventID), userID).Result()
		if err == redis.Nil {
			writeJSON(w, map[string]any{"status": "unknown"})
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		writeJSON(w, map[string]any{"status": "waiting", "position": rank + 1})
	}
}

func pathEventID(path, prefix, suffix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	eventID := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if eventID == "" {
		return "", false
	}
	return eventID, true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
