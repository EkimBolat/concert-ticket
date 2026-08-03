package main

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// admissionClaims is the payload of the short-lived JWT a user receives
// once the admitter lets them out of the queue. Downstream services
// (seat-locking, order) can verify this token to confirm the holder
// actually waited their turn.
type admissionClaims struct {
	EventID string `json:"eventId"`
	jwt.RegisteredClaims
}

func issueAdmissionToken(eventID, userID string, ttl time.Duration, secret []byte) (string, error) {
	now := time.Now()
	claims := admissionClaims{
		EventID: eventID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}
