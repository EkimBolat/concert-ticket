//go:build integration

// These tests hit a real Postgres instance (DATABASE_URL). The Seat
// Locking and Payment services are faked with httptest servers so the
// saga's branching logic can be tested without running the whole stack.
// Run them with:
//
//	go test -tags=integration ./...

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func jsonServer(handler func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}

func always(body map[string]any) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}
}

const testSecret = "test-internal-secret"

func testDeps(t *testing.T, paymentURL, seatLockingURL string) *dependencies {
	t.Helper()
	db := newDB()
	t.Cleanup(func() { db.Close() })
	return &dependencies{
		db:             db,
		seatLockingURL: seatLockingURL,
		paymentURL:     paymentURL,
		internalSecret: testSecret,
		events:         &eventPublisher{}, // nil channel -> publish() is a no-op
	}
}

func TestPlaceOrder_HappyPath(t *testing.T) {
	var paymentSecret, seatLockingSecret string

	payment := jsonServer(func(w http.ResponseWriter, r *http.Request) {
		paymentSecret = r.Header.Get("X-Internal-Secret")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "succeeded"})
	})
	defer payment.Close()

	seatLocking := jsonServer(func(w http.ResponseWriter, r *http.Request) {
		seatLockingSecret = r.Header.Get("X-Internal-Secret")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"confirmed": true})
	})
	defer seatLocking.Close()

	deps := testDeps(t, payment.URL, seatLocking.URL)

	result, err := placeOrder(deps, placeOrderRequest{
		EventID: "test-event", SeatID: "T1", UserID: "test-user", AmountCents: 1000,
	})
	if err != nil {
		t.Fatalf("placeOrder returned error: %v", err)
	}
	if result.Status != "CONFIRMED" {
		t.Fatalf("expected CONFIRMED, got %s", result.Status)
	}

	// The order service is the only trusted caller of these endpoints --
	// confirm this secret actually goes out on the wire, not just that
	// postJSON accepts a parameter for it.
	if paymentSecret != testSecret {
		t.Fatalf("expected payment charge to carry the internal secret, got %q", paymentSecret)
	}
	if seatLockingSecret != testSecret {
		t.Fatalf("expected seat-locking confirm to carry the internal secret, got %q", seatLockingSecret)
	}
}

func TestPlaceOrder_PaymentFails_ReleasesSeat(t *testing.T) {
	payment := jsonServer(always(map[string]any{"status": "failed"}))
	defer payment.Close()

	var seatLockingCalls []string
	seatLocking := jsonServer(func(w http.ResponseWriter, r *http.Request) {
		seatLockingCalls = append(seatLockingCalls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"released": true})
	})
	defer seatLocking.Close()

	deps := testDeps(t, payment.URL, seatLocking.URL)

	result, err := placeOrder(deps, placeOrderRequest{
		EventID: "test-event", SeatID: "T2", UserID: "test-user", AmountCents: 1000, SimulateFailure: true,
	})
	if err != nil {
		t.Fatalf("placeOrder returned error: %v", err)
	}
	if result.Status != "FAILED" || result.Reason != "payment_failed" {
		t.Fatalf("expected FAILED/payment_failed, got %s/%s", result.Status, result.Reason)
	}

	found := false
	for _, p := range seatLockingCalls {
		if p == "/seats/test-event/T2/release" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the seat-locking release endpoint to be called, calls were: %v", seatLockingCalls)
	}
}

func TestPlaceOrder_SeatConfirmFails_RefundsPayment(t *testing.T) {
	var refundCalled bool
	payment := jsonServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/refund" {
			refundCalled = true
			json.NewEncoder(w).Encode(map[string]any{"refunded": true})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"status": "succeeded"})
	})
	defer payment.Close()

	// Seat confirm always fails, to force the saga down the refund path.
	seatLocking := jsonServer(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "seat-locking unavailable", http.StatusInternalServerError)
	})
	defer seatLocking.Close()

	deps := testDeps(t, payment.URL, seatLocking.URL)

	result, err := placeOrder(deps, placeOrderRequest{
		EventID: "test-event", SeatID: "T3", UserID: "test-user", AmountCents: 1000,
	})
	if err != nil {
		t.Fatalf("placeOrder returned error: %v", err)
	}
	if result.Status != "FAILED" || result.Reason != "seat_confirm_failed" {
		t.Fatalf("expected FAILED/seat_confirm_failed, got %s/%s", result.Status, result.Reason)
	}
	if !refundCalled {
		t.Fatalf("expected the payment refund endpoint to be called as a compensating action")
	}
}
