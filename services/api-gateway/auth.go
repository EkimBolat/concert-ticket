package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// admissionClaims mirrors the claims the Waiting Room Service signs into
// its admission tokens (see services/waiting-room/token.go). The gateway
// needs to parse the same shape to verify them.
type admissionClaims struct {
	EventID string `json:"eventId"`
	jwt.RegisteredClaims
}

// identityExtractor pulls the eventId/userId a request is claiming to act
// on, from wherever they live for that particular route (URL path, JSON
// body, or both). Returns ok=false if the request doesn't look like a
// well-formed request for that route.
type identityExtractor func(r *http.Request, body []byte) (eventID, userID string, ok bool)

// requireAdmission is the whole point of this file: without it, nothing
// stops a client from skipping the Waiting Room entirely and hitting
// Seat Locking / Order directly. It verifies the admission JWT's
// signature and expiry, then checks that the token was actually issued
// for the exact event and user the request claims to be.
func requireAdmission(secret []byte, extract identityExtractor, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, ok := bearerToken(r)
		if !ok {
			http.Error(w, "missing or malformed Authorization header (expected: Bearer <admission token>)", http.StatusUnauthorized)
			return
		}

		claims := &admissionClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
			return secret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid or expired admission token -- join the waiting room first", http.StatusUnauthorized)
			return
		}

		// Read the body to check it against the token, then put it back
		// so the reverse proxy can still forward it downstream.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))

		eventID, userID, ok := extract(r, body)
		if !ok {
			http.Error(w, "could not determine eventId/userId for this request", http.StatusBadRequest)
			return
		}

		if claims.EventID != eventID || claims.Subject != userID {
			http.Error(w, "admission token does not match this event/user", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	return strings.TrimPrefix(header, prefix), true
}

// lockIdentity reads eventId from the URL path (/seats/{eventId}/{seatId}/lock)
// and userId from the JSON body ({"userId": "..."}).
func lockIdentity(r *http.Request, body []byte) (eventID, userID string, ok bool) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "seats" || parts[3] != "lock" {
		return "", "", false
	}

	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.UserID == "" {
		return "", "", false
	}
	return parts[1], payload.UserID, true
}

// orderIdentity reads both eventId and userId from the JSON body of a
// POST /orders request.
func orderIdentity(r *http.Request, body []byte) (eventID, userID string, ok bool) {
	var payload struct {
		EventID string `json:"eventId"`
		UserID  string `json:"userId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.EventID == "" || payload.UserID == "" {
		return "", "", false
	}
	return payload.EventID, payload.UserID, true
}
