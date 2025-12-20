package subnetupdatepublisher

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisPublish_SendsReloadMessage(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis run: %v", err)
	}
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	pub, err := NewRedisSubnetUpdatesPublisher(rdb, "abf:subnets")
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}

	// subscribe to channel to observe message
	sub := rdb.Subscribe(context.Background(), "abf:subnets")
	defer sub.Close()
	ch := sub.Channel()

	if err := pub.PublishSubnetUpdated(context.Background()); err != nil {
		t.Fatalf("publish error: %v", err)
	}

	select {
	case msg := <-ch:
		if msg.Payload != "reload" {
			t.Fatalf("unexpected payload: %q", msg.Payload)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for published message")
	}
}
