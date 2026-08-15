package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// TestSeatsHandler_ConfirmAndReleaseRequireInternalSecret guards the fix
// for the bug where any client that could reach this service -- through
// the gateway's blanket /seats/ passthrough, or directly -- could call
// confirm/release itself and mark a seat SOLD (or release someone
// else's checkout) without ever going through the order/payment saga.
// It doesn't need a real Redis: a rejected request never reaches it.
func TestSeatsHandler_ConfirmAndReleaseRequireInternalSecret(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	handler := seatsHandler(rdb, "the-real-secret")

	for _, action := range []string{"release", "confirm"} {
		t.Run(action+"/no secret", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/seats/evt/S1/"+action, strings.NewReader(`{"userId":"u1"}`))
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403 with no secret, got %d", rec.Code)
			}
		})

		t.Run(action+"/wrong secret", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/seats/evt/S1/"+action, strings.NewReader(`{"userId":"u1"}`))
			req.Header.Set("X-Internal-Secret", "not-the-secret")
			rec := httptest.NewRecorder()
			handler(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("expected 403 with wrong secret, got %d", rec.Code)
			}
		})
	}
}
