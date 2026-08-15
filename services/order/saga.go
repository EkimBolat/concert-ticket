package main

import "log"

type placeOrderRequest struct {
	EventID         string `json:"eventId"`
	SeatID          string `json:"seatId"`
	UserID          string `json:"userId"`
	AmountCents     int64  `json:"amountCents"`
	SimulateFailure bool   `json:"simulateFailure"`
}

type placeOrderResult struct {
	OrderID int64  `json:"orderId"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
}

// placeOrder runs the purchase saga: charge payment -> confirm the seat,
// with a compensating action for whichever step fails. This function is
// the whole point of the Order Service.
func placeOrder(deps *dependencies, req placeOrderRequest) (placeOrderResult, error) {
	orderID, err := insertOrder(deps.db, req.EventID, req.SeatID, req.UserID, req.AmountCents, "PENDING")
	if err != nil {
		return placeOrderResult{}, err
	}

	// Step 1: charge payment.
	paymentStatus, err := chargePayment(deps.paymentURL, orderID, req.UserID, req.AmountCents, req.SimulateFailure, deps.internalSecret)
	if err != nil || paymentStatus != "succeeded" {
		log.Printf("order %d: payment failed (err=%v status=%q), releasing seat", orderID, err, paymentStatus)
		// Compensating action: we never took the money, so give the seat back.
		if relErr := releaseSeat(deps.seatLockingURL, req.EventID, req.SeatID, req.UserID, deps.internalSecret); relErr != nil {
			log.Printf("order %d: failed to release seat during compensation: %v", orderID, relErr)
		}
		_ = updateOrderStatus(deps.db, orderID, "FAILED")
		deps.events.publish("order.failed", map[string]any{"orderId": orderID, "reason": "payment_failed"})
		return placeOrderResult{OrderID: orderID, Status: "FAILED", Reason: "payment_failed"}, nil
	}

	// Step 2: confirm the seat as sold.
	if err := confirmSeat(deps.seatLockingURL, req.EventID, req.SeatID, req.UserID, deps.internalSecret); err != nil {
		log.Printf("order %d: seat confirm failed (%v), refunding payment", orderID, err)
		// Compensating action: payment succeeded but we couldn't keep the
		// seat, so give the money back.
		if refErr := refundPayment(deps.paymentURL, orderID, deps.internalSecret); refErr != nil {
			log.Printf("order %d: failed to refund during compensation: %v", orderID, refErr)
		}
		_ = updateOrderStatus(deps.db, orderID, "FAILED")
		deps.events.publish("order.failed", map[string]any{"orderId": orderID, "reason": "seat_confirm_failed"})
		return placeOrderResult{OrderID: orderID, Status: "FAILED", Reason: "seat_confirm_failed"}, nil
	}

	_ = updateOrderStatus(deps.db, orderID, "CONFIRMED")
	deps.events.publish("order.completed", map[string]any{
		"orderId": orderID, "eventId": req.EventID, "seatId": req.SeatID, "userId": req.UserID,
	})
	return placeOrderResult{OrderID: orderID, Status: "CONFIRMED"}, nil
}
