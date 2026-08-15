package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// 60s, not a more typical 5s -- the seat-locking and payment services are
// hosted on Render's free tier, which spins instances down when idle and
// can take up to ~50s to wake one back up on the next request.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// postJSON sends the internal secret on every call -- seat-locking's
// release/confirm and payment's charge/refund are all restricted to
// server-to-server callers, and the order service is the only caller
// those endpoints trust.
func postJSON(url string, body any, secret string, out any) (int, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Secret", secret)

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

// --- Seat Locking client ---

func confirmSeat(baseURL, eventID, seatID, userID, secret string) error {
	url := fmt.Sprintf("%s/seats/%s/%s/confirm", baseURL, eventID, seatID)
	var out struct {
		Confirmed bool `json:"confirmed"`
	}
	status, err := postJSON(url, map[string]string{"userId": userID}, secret, &out)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("seat-locking confirm failed: status %d", status)
	}
	if !out.Confirmed {
		return fmt.Errorf("seat-locking confirm refused: seat is not locked by this user")
	}
	return nil
}

func releaseSeat(baseURL, eventID, seatID, userID, secret string) error {
	url := fmt.Sprintf("%s/seats/%s/%s/release", baseURL, eventID, seatID)
	var out map[string]any
	status, err := postJSON(url, map[string]string{"userId": userID}, secret, &out)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("seat-locking release failed: status %d", status)
	}
	return nil
}

// --- Payment client ---

type chargeResult struct {
	Status string `json:"status"`
}

func chargePayment(baseURL string, orderID int64, userID string, amountCents int64, simulateFailure bool, secret string) (string, error) {
	url := baseURL + "/charge"
	var out chargeResult
	body := map[string]any{
		"orderId":         orderID,
		"userId":          userID,
		"amountCents":     amountCents,
		"simulateFailure": simulateFailure,
	}
	status, err := postJSON(url, body, secret, &out)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("payment charge failed: status %d", status)
	}
	return out.Status, nil
}

func refundPayment(baseURL string, orderID int64, secret string) error {
	url := baseURL + "/refund"
	var out map[string]any
	status, err := postJSON(url, map[string]any{"orderId": orderID}, secret, &out)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("payment refund failed: status %d", status)
	}
	return nil
}
