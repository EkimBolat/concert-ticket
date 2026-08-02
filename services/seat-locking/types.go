package main

import "time"

// SeatEvent is published on Redis Pub/Sub whenever a seat's state changes,
// and forwarded verbatim to every WebSocket client watching that event.
type SeatEvent struct {
	EventID  string    `json:"eventId"`
	SeatID   string    `json:"seatId"`
	Status   string    `json:"status"` // "locked" | "released" | "sold"
	LockedBy string    `json:"lockedBy,omitempty"`
	At       time.Time `json:"at"`
}

const (
	StatusLocked   = "locked"
	StatusReleased = "released"
	StatusSold     = "sold"
)
