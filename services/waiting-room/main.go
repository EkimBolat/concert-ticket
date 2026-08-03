package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func main() {
	port := getenv("PORT", "8081")
	secret := []byte(getenv("JWT_SECRET", "dev-secret-change-me"))
	if string(secret) == "dev-secret-change-me" {
		log.Println("warning: using default JWT_SECRET, set a real one before deploying")
	}

	batchSize := getenvInt("ADMIT_BATCH_SIZE", 5)
	admitInterval := time.Duration(getenvInt("ADMIT_INTERVAL_SECONDS", 10)) * time.Second
	tokenTTL := time.Duration(getenvInt("TOKEN_TTL_MINUTES", 10)) * time.Minute

	rdb := newRedisClient()
	startAdmitter(rdb, batchSize, admitInterval, tokenTTL, secret)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"service": "waiting-room", "status": "ok"})
	})

	// POST /queue/{eventId}/join
	// GET  /queue/{eventId}/status?userId=...
	mux.HandleFunc("/queue/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/join"):
			joinHandler(rdb)(w, r)
		case strings.HasSuffix(r.URL.Path, "/status"):
			statusHandler(rdb)(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	log.Printf("waiting-room service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
