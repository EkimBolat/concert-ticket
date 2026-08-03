package main

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

// activeQueuesKey holds the set of eventIDs that currently have someone
// waiting, so the admitter only scans events that actually need it.
const activeQueuesKey = "active-queues"

func newRedisClient() *redis.Client {
	addr := getenv("REDIS_ADDR", "localhost:6379")
	return redis.NewClient(&redis.Options{Addr: addr})
}

func queueKey(eventID string) string {
	return fmt.Sprintf("queue:%s", eventID)
}

func admittedKey(eventID, userID string) string {
	return fmt.Sprintf("admitted:%s:%s", eventID, userID)
}
