package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	lockTTL       = 5 * time.Minute
	pubSubChannel = "seat-updates" // actual channel is "seat-updates:{eventId}"
)

func newRedisClient() *redis.Client {
	addr := getenv("REDIS_ADDR", "localhost:6379")
	return redis.NewClient(&redis.Options{Addr: addr})
}

func seatKey(eventID, seatID string) string {
	return fmt.Sprintf("seat:%s:%s", eventID, seatID)
}

func channelFor(eventID string) string {
	return fmt.Sprintf("%s:%s", pubSubChannel, eventID)
}

// lockSeat is the core concurrency guarantee of the whole project: SETNX
// only succeeds for the first caller, so two simultaneous requests for the
// same seat can never both win. The TTL means an abandoned checkout
// releases the seat automatically, no cleanup job required.
func lockSeat(ctx context.Context, rdb *redis.Client, eventID, seatID, userID string) (bool, error) {
	key := seatKey(eventID, seatID)
	ok, err := rdb.SetNX(ctx, key, userID, lockTTL).Result()
	if err != nil {
		return false, err
	}
	if ok {
		publish(ctx, rdb, eventID, SeatEvent{
			EventID: eventID, SeatID: seatID,
			Status: StatusLocked, LockedBy: userID, At: time.Now(),
		})
	}
	return ok, nil
}

// releaseSeat is the saga's compensating action: it only deletes the lock
// if it's still held by the same user who acquired it, so it can't
// accidentally release a lock that someone else has since acquired.
func releaseSeat(ctx context.Context, rdb *redis.Client, eventID, seatID, userID string) (bool, error) {
	key := seatKey(eventID, seatID)
	current, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // nothing to release
	}
	if err != nil {
		return false, err
	}
	if current != userID {
		return false, nil // held by someone else, refuse
	}
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return false, err
	}
	publish(ctx, rdb, eventID, SeatEvent{
		EventID: eventID, SeatID: seatID,
		Status: StatusReleased, At: time.Now(),
	})
	return true, nil
}

// confirmSeat is called by the Order Service once payment succeeds. It
// removes the TTL so the seat is permanently SOLD instead of expiring.
func confirmSeat(ctx context.Context, rdb *redis.Client, eventID, seatID, userID string) error {
	key := seatKey(eventID, seatID)
	if err := rdb.Persist(ctx, key).Err(); err != nil {
		return err
	}
	publish(ctx, rdb, eventID, SeatEvent{
		EventID: eventID, SeatID: seatID,
		Status: StatusSold, LockedBy: userID, At: time.Now(),
	})
	return nil
}

func publish(ctx context.Context, rdb *redis.Client, eventID string, evt SeatEvent) {
	b, err := json.Marshal(evt)
	if err != nil {
		return
	}
	rdb.Publish(ctx, channelFor(eventID), b)
}
