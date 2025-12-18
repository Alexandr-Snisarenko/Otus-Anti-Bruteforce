package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/memory"
)

func TestLimiter_BasicFlow(t *testing.T) {
	tests := []struct {
		name        string
		rule        Rule
		attempts    int // number of attempts to make
		wantAllow   bool
		wantAllowed int // number of attempts that should be allowed
	}{
		{
			name: "under limit",
			rule: Rule{
				Limit:  3,
				Window: time.Second,
			},
			attempts:    2,
			wantAllow:   true,
			wantAllowed: 2,
		},
		{
			name: "at limit",
			rule: Rule{
				Limit:  3,
				Window: time.Second,
			},
			attempts:    3,
			wantAllow:   true,
			wantAllowed: 3,
		},
		{
			name: "over limit",
			rule: Rule{
				Limit:  3,
				Window: time.Second,
			},
			attempts:    5,
			wantAllow:   false, // last attempt should be rejected
			wantAllowed: 3,
		},
		{
			name: "zero limit",
			rule: Rule{
				Limit:  0,
				Window: time.Second,
			},
			attempts:    1,
			wantAllow:   false,
			wantAllowed: 0,
		},
		{
			name: "zero window",
			rule: Rule{
				Limit:  5,
				Window: 0,
			},
			attempts:    10,
			wantAllow:   true, // per inmemorydb implementation, zero window = always allow
			wantAllowed: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := memory.NewBucketsDB()
			key := "test-key"
			l := NewLimiter(db, "test-bucket", tt.rule)
			ctx := context.Background()

			allowed := 0
			var lastAllow bool
			var lastErr error

			// Make attempts
			for i := 0; i < tt.attempts; i++ {
				lastAllow, lastErr = l.Allow(ctx, key)
				if lastErr != nil {
					t.Fatalf("attempt %d: unexpected error: %v", i+1, lastErr)
				}
				if lastAllow {
					allowed++
				}
			}

			if lastAllow != tt.wantAllow {
				t.Errorf("last attempt: got allow=%v, want %v", lastAllow, tt.wantAllow)
			}
			if allowed != tt.wantAllowed {
				t.Errorf("total allowed: got %d, want %d", allowed, tt.wantAllowed)
			}
		})
	}
}

func TestLimiter_SlidingWindow(t *testing.T) {
	db := memory.NewBucketsDB()
	rule := Rule{
		Limit:  2,
		Window: 100 * time.Millisecond,
	}
	key := "sliding-key"
	l := NewLimiter(db, "sliding-bucket", rule)
	ctx := context.Background()

	// First attempt should be allowed
	allow, err := l.Allow(ctx, key)
	if err != nil || !allow {
		t.Fatalf("first attempt: got allow=%v err=%v, want allow=true err=nil", allow, err)
	}

	// Second attempt should be allowed
	allow, err = l.Allow(ctx, key)
	if err != nil || !allow {
		t.Fatalf("second attempt: got allow=%v err=%v, want allow=true err=nil", allow, err)
	}

	// Third attempt should be rejected (limit reached)
	allow, err = l.Allow(ctx, key)
	if err != nil || allow {
		t.Fatalf("third attempt: got allow=%v err=%v, want allow=false err=nil", allow, err)
	}

	// Wait for window to slide past first attempt
	time.Sleep(rule.Window + 10*time.Millisecond)

	// Should allow one more attempt now
	allow, err = l.Allow(ctx, key)
	if err != nil || !allow {
		t.Fatalf("after window: got allow=%v err=%v, want allow=true err=nil", allow, err)
	}
}

func TestLimiter_Reset(t *testing.T) {
	db := memory.NewBucketsDB()
	rule := Rule{
		Limit:  2,
		Window: time.Minute,
	}
	key := "reset-key"
	l := NewLimiter(db, "reset-bucket", rule)
	ctx := context.Background()

	// Up the limit
	for i := 0; i < rule.Limit; i++ {
		allow, err := l.Allow(ctx, key)
		if err != nil || !allow {
			t.Fatalf("attempt %d: got allow=%v err=%v, want allow=true err=nil", i+1, allow, err)
		}
	}

	// Next attempt should be rejected
	if allow, err := l.Allow(ctx, key); err != nil || allow {
		t.Fatalf("over-limit: got allow=%v err=%v, want allow=false err=nil", allow, err)
	}

	// Reset should clear the limit
	if err := l.Reset(ctx, key); err != nil {
		t.Fatalf("reset: unexpected error: %v", err)
	}

	// Should be allowed again after reset
	if allow, err := l.Allow(ctx, key); err != nil || !allow {
		t.Fatalf("after reset: got allow=%v err=%v, want allow=true err=nil", allow, err)
	}
}

func TestLimiter_MultipleKeysIsolation(t *testing.T) {
	db := memory.NewBucketsDB()
	rule := Rule{
		Limit:  2,
		Window: time.Second,
	}

	// Create two limiters with different buckets
	key := "multi-key"
	l1 := NewLimiter(db, "bucket1", rule)
	l2 := NewLimiter(db, "bucket2", rule)
	ctx := context.Background()

	// Up first limiter
	for i := 0; i < rule.Limit; i++ {
		allow, err := l1.Allow(ctx, key)
		if err != nil || !allow {
			t.Fatalf("l1 attempt %d: got allow=%v err=%v, want allow=true err=nil", i+1, allow, err)
		}
	}

	// First limiter should be blocked
	if allow, err := l1.Allow(ctx, key); err != nil || allow {
		t.Fatalf("l1 over-limit: got allow=%v err=%v, want allow=false err=nil", allow, err)
	}

	// Second limiter should still allow attempts
	for i := 0; i < rule.Limit; i++ {
		allow, err := l2.Allow(ctx, key)
		if err != nil || !allow {
			t.Fatalf("l2 attempt %d: got allow=%v err=%v, want allow=true err=nil", i+1, allow, err)
		}
	}
}
