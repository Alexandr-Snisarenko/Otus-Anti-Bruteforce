package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	mem "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/memory"
)

func TestLimitChecker_AllowAndReset(t *testing.T) {
	storage := mem.NewBucketsDB()
	limits := Limits{
		domain.LoginLimit: Rule{Limit: 3, Window: 200 * time.Millisecond},
	}
	lc := NewLimitChecker(storage, limits)

	ctx := context.Background()
	key := "user1"

	// first three attempts allowed
	for i := 0; i < 3; i++ {
		ok, err := lc.Allow(ctx, domain.LoginLimit, key)
		if err != nil {
			t.Fatalf("Allow returned error: %v", err)
		}
		if !ok {
			t.Fatalf("expected allow on attempt %d", i+1)
		}
	}

	// 4th attempt should be denied
	ok, err := lc.Allow(ctx, domain.LoginLimit, key)
	if err != nil {
		t.Fatalf("Allow returned error: %v", err)
	}
	if ok {
		t.Fatalf("expected 4th attempt to be denied")
	}

	// reset and expect allow again
	if err := lc.Reset(ctx, domain.LoginLimit, key); err != nil {
		t.Fatalf("Reset returned error: %v", err)
	}
	ok, err = lc.Allow(ctx, domain.LoginLimit, key)
	if err != nil {
		t.Fatalf("Allow after reset returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected allow after reset")
	}
}

func TestLimitChecker_WindowExpiry(t *testing.T) {
	storage := mem.NewBucketsDB()
	limits := Limits{
		domain.LoginLimit: Rule{Limit: 2, Window: 100 * time.Millisecond},
	}
	lc := NewLimitChecker(storage, limits)

	ctx := context.Background()
	key := "user2"

	// two attempts allowed
	if ok, _ := lc.Allow(ctx, domain.LoginLimit, key); !ok {
		t.Fatalf("expected first attempt allowed")
	}
	if ok, _ := lc.Allow(ctx, domain.LoginLimit, key); !ok {
		t.Fatalf("expected second attempt allowed")
	}

	// third attempt denied
	if ok, _ := lc.Allow(ctx, domain.LoginLimit, key); ok {
		t.Fatalf("expected third attempt denied")
	}

	// wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// now should be allowed again
	if ok, _ := lc.Allow(ctx, domain.LoginLimit, key); !ok {
		t.Fatalf("expected attempt allowed after window expiry")
	}
}

func TestLimitChecker_MissingTypeAndZeroLimits(t *testing.T) {
	storage := mem.NewBucketsDB()
	limits := Limits{
		domain.LoginLimit: Rule{Limit: 0, Window: time.Second},
	}
	lc := NewLimitChecker(storage, limits)

	ctx := context.Background()

	// missing type -> returns false,nil
	ok, err := lc.Allow(ctx, domain.PasswordLimit, "any")
	if err != nil {
		t.Fatalf("unexpected error for missing type: %v", err)
	}
	if ok {
		t.Fatalf("expected false for missing limit type")
	}

	// zero limit configured -> should be denied
	ok, err = lc.Allow(ctx, domain.LoginLimit, "k")
	if err != nil {
		t.Fatalf("unexpected error for zero limit: %v", err)
	}
	if ok {
		t.Fatalf("expected denied when limit is zero")
	}

	// Reset on missing type should not error
	if err := lc.Reset(ctx, domain.PasswordLimit, "k"); err != nil {
		t.Fatalf("Reset on missing type returned error: %v", err)
	}
}
