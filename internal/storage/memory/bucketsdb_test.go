package memory

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAllow_BasicSlidingWindow(t *testing.T) {
	db := NewBucketsDB()
	ctx := context.Background()

	// limit 3 attempts per 100ms window
	limit := 3
	window := 100 * time.Millisecond

	key := "user1"

	// first three should be allowed
	for i := 0; i < limit; i++ {
		ok, err := db.Allow(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatalf("expected allowed on attempt %d", i+1)
		}
	}

	// fourth should be rejected
	ok, _ := db.Allow(ctx, key, limit, window)
	if ok {
		t.Fatalf("expected rejected on attempt %d", limit+1)
	}

	// wait for window to expire, then it should be allowed again
	time.Sleep(window + 10*time.Millisecond)
	ok, _ = db.Allow(ctx, key, limit, window)
	if !ok {
		t.Fatalf("expected allowed after window expiration")
	}
}

func TestAllow_EdgeCases(t *testing.T) {
	db := NewBucketsDB()
	ctx := context.Background()

	// limit <= 0 -> always false
	ok, _ := db.Allow(ctx, "k1", 0, time.Second)
	if ok {
		t.Fatalf("expected not allowed when limit <= 0")
	}

	// window <= 0 -> always true (per implementation)
	// even if limit is 0 or small, window<=0 returns true
	ok, _ = db.Allow(ctx, "k1", 1, 0)
	if !ok {
		t.Fatalf("expected allowed when window <= 0")
	}
}

func TestReset_ClearsTimeline(t *testing.T) {
	db := NewBucketsDB()
	ctx := context.Background()

	key := "reset-key"
	limit := 2
	window := time.Second

	// consume limit
	for i := 0; i < limit; i++ {
		ok, _ := db.Allow(ctx, key, limit, window)
		if !ok {
			t.Fatalf("unexpected rejected before reset")
		}
	}

	// next should be rejected
	ok, _ := db.Allow(ctx, key, limit, window)
	if ok {
		t.Fatalf("expected rejected before reset")
	}

	// reset and allow again
	if err := db.Reset(ctx, key); err != nil {
		t.Fatalf("reset returned error: %v", err)
	}
	ok, _ = db.Allow(ctx, key, limit, window)
	if !ok {
		t.Fatalf("expected allowed after reset")
	}
}

func TestAllow_Concurrency(t *testing.T) {
	db := NewBucketsDB()
	ctx := context.Background()

	key := "concurrent"
	limit := 100
	window := time.Second

	var wg sync.WaitGroup
	n := 500
	var allowed int32

	// spawn many goroutines calling Allow concurrently
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ok, _ := db.Allow(ctx, key, limit, window)
			if ok {
				atomic.AddInt32(&allowed, 1)
			}
		}()
	}
	wg.Wait()

	if atomic.LoadInt32(&allowed) > int32(limit) {
		t.Fatalf("concurrent allowed count %d exceeds limit %d", allowed, limit)
	}

	// Because of the concurrency semantics, the number of allowed requests
	// must not exceed the limit. We'll exercise that by running another
	// sequential check: try limit times and ensure at most limit are allowed
	// after resetting
	if err := db.Reset(ctx, key); err != nil {
		t.Fatalf("reset returned error: %v", err)
	}

	got := 0
	for i := 0; i < limit*2; i++ {
		ok, _ := db.Allow(ctx, key, limit, window)
		if ok {
			got++
		}
	}
	if got != limit {
		t.Fatalf("expected exactly %d allowed sequentially after reset, got %d", limit, got)
	}
}
