package main

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// startAdmitter runs forever in the background, letting a batch of users
// out of each active queue every `interval`. This is what makes the
// waiting room a *throttle* instead of just a list: no matter how many
// thousands of people are queued, only `batchSize` of them move forward
// per tick.
func startAdmitter(rdb *redis.Client, batchSize int64, interval time.Duration, tokenTTL time.Duration, secret []byte) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			admitTick(context.Background(), rdb, batchSize, tokenTTL, secret)
		}
	}()
}

func admitTick(ctx context.Context, rdb *redis.Client, batchSize int64, tokenTTL time.Duration, secret []byte) {
	eventIDs, err := rdb.SMembers(ctx, activeQueuesKey).Result()
	if err != nil {
		log.Printf("admitter: failed to list active queues: %v", err)
		return
	}

	for _, eventID := range eventIDs {
		admitBatch(ctx, rdb, eventID, batchSize, tokenTTL, secret)
	}
}

func admitBatch(ctx context.Context, rdb *redis.Client, eventID string, batchSize int64, tokenTTL time.Duration, secret []byte) {
	// ZPopMin removes and returns the lowest-score (earliest arrival)
	// members atomically — this is the actual "let the next N people in"
	// operation.
	popped, err := rdb.ZPopMin(ctx, queueKey(eventID), batchSize).Result()
	if err != nil {
		log.Printf("admitter: failed to pop queue for %s: %v", eventID, err)
		return
	}
	if len(popped) == 0 {
		// Nobody waiting — stop scanning this event until someone joins again.
		rdb.SRem(ctx, activeQueuesKey, eventID)
		return
	}

	for _, z := range popped {
		userID, ok := z.Member.(string)
		if !ok {
			continue
		}
		token, err := issueAdmissionToken(eventID, userID, tokenTTL, secret)
		if err != nil {
			log.Printf("admitter: failed to issue token for %s/%s: %v", eventID, userID, err)
			continue
		}
		if err := rdb.Set(ctx, admittedKey(eventID, userID), token, tokenTTL).Err(); err != nil {
			log.Printf("admitter: failed to store token for %s/%s: %v", eventID, userID, err)
			continue
		}
		log.Printf("admitter: admitted user=%s event=%s", userID, eventID)
	}
}
