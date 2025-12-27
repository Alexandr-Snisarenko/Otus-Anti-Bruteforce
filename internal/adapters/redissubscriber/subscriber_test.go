package redissubscriber

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type fakeHolder struct {
	calls int32
	err   error
}

func (f *fakeHolder) ReloadSubnets(_ context.Context) error {
	atomic.AddInt32(&f.calls, 1)
	return f.err
}

func TestSubscriber_ReloadsOnMessageAndStopsOnContextCancel(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	fh := &fakeHolder{}
	sub := NewSubnetUpdatesSubscriber(rdb, fh, "abf:subnets")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- sub.Start(ctx)
	}()

	// wait until subscription is registered
	waitSub := time.After(500 * time.Millisecond)
	for {
		m, err := rdb.PubSubNumSub(context.Background(), "abf:subnets").Result()
		if err == nil {
			if v, ok := m["abf:subnets"]; ok && v > 0 {
				break
			}
		}
		select {
		case <-waitSub:
			t.Fatal("timeout waiting for subscription to be registered")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// publish a message and expect ReloadSubnets to be called
	if err := rdb.Publish(context.Background(), "abf:subnets", "reload").Err(); err != nil {
		t.Fatalf("publish error: %v", err)
	}

	// wait until the fake holder registers a call
	wait := time.After(1 * time.Second)
	for {
		if atomic.LoadInt32(&fh.calls) > 0 {
			break
		}
		select {
		case <-wait:
			t.Fatal("timeout waiting for ReloadSubnets call")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// cancel context and expect Start to exit with ctx.Err()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && err != nil {
			t.Fatalf("expected context cancellation, got: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Start to exit after cancel")
	}
}

func TestSubscriber_PropagatesReloadError(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	fh := &fakeHolder{err: errors.New("reload fail")}
	sub := NewSubnetUpdatesSubscriber(rdb, fh, "abf:subnets")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- sub.Start(ctx)
	}()

	// wait until subscription is registered
	waitSub := time.After(500 * time.Millisecond)
	for {
		m, err := rdb.PubSubNumSub(context.Background(), "abf:subnets").Result()
		if err == nil {
			if v, ok := m["abf:subnets"]; ok && v > 0 {
				break
			}
		}
		select {
		case <-waitSub:
			t.Fatal("timeout waiting for subscription to be registered")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	if err := rdb.Publish(context.Background(), "abf:subnets", "reload").Err(); err != nil {
		t.Fatalf("publish error: %v", err)
	}

	select {
	case err := <-done:
		if err == nil || err.Error() != "reload subnet policy: reload fail" {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Start to return after reload error")
	}
	cancel()
}

func TestSubscriber_ChannelClosedReturnsError(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	fh := &fakeHolder{}
	sub := NewSubnetUpdatesSubscriber(rdb, fh, "abf:subnets")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- sub.Start(ctx)
	}()

	// wait until subscription is registered
	waitSub2 := time.After(500 * time.Millisecond)
	for {
		m, err := rdb.PubSubNumSub(context.Background(), "abf:subnets").Result()
		if err == nil {
			if v, ok := m["abf:subnets"]; ok && v > 0 {
				break
			}
		}
		select {
		case <-waitSub2:
			t.Fatal("timeout waiting for subscription to be registered")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Close the redis client which should close the pubsub channel
	if err := rdb.Close(); err != nil {
		t.Fatalf("rdb close: %v", err)
	}

	select {
	case err := <-done:
		if err == nil || err.Error() != "redis pubsub channel closed" {
			t.Fatalf("unexpected error after channel closed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Start to return after channel close")
	}
	cancel()
}
