package redisdb

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *redis.Client, func()) {
	t.Helper()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cleanup := func() {
		client.Close()
		s.Close()
	}
	return s, client, cleanup
}

func TestBucketsRepo_AllowAndReset(t *testing.T) {
	s, client, cleanup := setupMiniredis(t)
	defer cleanup()

	repo := NewBucketsRepo(client)
	ctx := context.Background()
	key := "login:user1"
	limit := 3
	window := 1 * time.Second

	// first three attempts allowed
	for i := 0; i < limit; i++ {
		ok, err := repo.Allow(ctx, key, limit, window)
		if err != nil {
			t.Fatalf("Allow returned error: %v", err)
		}
		if !ok {
			t.Fatalf("expected allow at attempt %d", i+1)
		}
	}

	// next attempt should be denied
	ok, err := repo.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Allow returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected attempt to be denied after limit")
	}

	// Reset and expect allow again
	if err := repo.Reset(ctx, key); err != nil {
		t.Fatalf("Reset returned error: %v", err)
	}
	ok, err = repo.Allow(ctx, key, limit, window)
	if err != nil {
		t.Fatalf("Allow returned error after reset: %v", err)
	}
	if !ok {
		t.Fatalf("expected allow after reset")
	}

	// check that redis contains the key with TTL set
	if !s.Exists(key) {
		t.Fatalf("expected redis key %s to exist", key)
	}
}

func TestBucketsRepo_WindowExpiry(t *testing.T) {
	_, client, cleanup := setupMiniredis(t)
	defer cleanup()

	repo := NewBucketsRepo(client)
	ctx := context.Background()
	key := "login:user2"
	limit := 2
	window := 1 * time.Second

	// two allowed
	if ok, _ := repo.Allow(ctx, key, limit, window); !ok {
		t.Fatalf("expected first allowed")
	}
	if ok, _ := repo.Allow(ctx, key, limit, window); !ok {
		t.Fatalf("expected second allowed")
	}

	// third attempt denied
	if ok, _ := repo.Allow(ctx, key, limit, window); ok {
		t.Fatalf("expected third attempt denied")
	}

	// wait for window to pass
	time.Sleep(window + 20*time.Millisecond)

	// now should be allowed again
	if ok, _ := repo.Allow(ctx, key, limit, window); !ok {
		t.Fatalf("expected allowed after window expiry")
	}
}
