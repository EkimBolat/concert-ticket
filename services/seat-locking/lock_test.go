//go:build integration

// These tests hit a real Redis instance (REDIS_ADDR, default
// localhost:6379). Run them with:
//
//	go test -tags=integration ./...

package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func uniqueEventID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-event-%d", time.Now().UnixNano())
}

func TestLockSeat_SecondCallerIsRejected(t *testing.T) {
	rdb := newRedisClient()
	defer rdb.Close()
	ctx := context.Background()
	eventID := uniqueEventID(t)
	defer rdb.Del(ctx, seatKey(eventID, "S1"))

	ok1, err := lockSeat(ctx, rdb, eventID, "S1", "user-a")
	if err != nil {
		t.Fatalf("lockSeat error: %v", err)
	}
	if !ok1 {
		t.Fatalf("expected the first caller to acquire the lock")
	}

	ok2, err := lockSeat(ctx, rdb, eventID, "S1", "user-b")
	if err != nil {
		t.Fatalf("lockSeat error: %v", err)
	}
	if ok2 {
		t.Fatalf("expected the second caller to be rejected, but it acquired the lock")
	}
}

// TestLockSeat_ConcurrentCallers_OnlyOneWins is the whole point of the
// project: fire many concurrent requests at the same seat and prove that
// exactly one of them ever wins the lock.
func TestLockSeat_ConcurrentCallers_OnlyOneWins(t *testing.T) {
	rdb := newRedisClient()
	defer rdb.Close()
	ctx := context.Background()
	eventID := uniqueEventID(t)
	defer rdb.Del(ctx, seatKey(eventID, "S2"))

	const attempts = 50
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0

	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := lockSeat(ctx, rdb, eventID, "S2", fmt.Sprintf("user-%d", i))
			if err != nil {
				t.Errorf("lockSeat error: %v", err)
				return
			}
			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()

	if wins != 1 {
		t.Fatalf("expected exactly 1 of %d concurrent callers to win the lock, got %d", attempts, wins)
	}
}

func TestReleaseSeat_OnlyOwnerCanRelease(t *testing.T) {
	rdb := newRedisClient()
	defer rdb.Close()
	ctx := context.Background()
	eventID := uniqueEventID(t)
	defer rdb.Del(ctx, seatKey(eventID, "S3"))

	if _, err := lockSeat(ctx, rdb, eventID, "S3", "owner"); err != nil {
		t.Fatalf("lockSeat error: %v", err)
	}

	released, err := releaseSeat(ctx, rdb, eventID, "S3", "not-the-owner")
	if err != nil {
		t.Fatalf("releaseSeat error: %v", err)
	}
	if released {
		t.Fatalf("expected release by a non-owner to be refused")
	}

	released, err = releaseSeat(ctx, rdb, eventID, "S3", "owner")
	if err != nil {
		t.Fatalf("releaseSeat error: %v", err)
	}
	if !released {
		t.Fatalf("expected release by the owner to succeed")
	}
}
