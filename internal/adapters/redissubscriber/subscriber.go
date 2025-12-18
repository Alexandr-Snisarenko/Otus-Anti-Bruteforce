package redissubscriber

import (
	"context"
	"fmt"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ports"
	"github.com/redis/go-redis/v9"
)

type SubnetUpdatesSubscriber struct {
	rdb          *redis.Client
	channel      string
	subnetHolder ports.SubnetHolder
}

func NewSubnetUpdatesSubscriber(
	rdb *redis.Client,
	subnetHolder ports.SubnetHolder,
	channel string,
) *SubnetUpdatesSubscriber {
	return &SubnetUpdatesSubscriber{rdb: rdb, channel: channel, subnetHolder: subnetHolder}
}

func (s *SubnetUpdatesSubscriber) Start(ctx context.Context) error {
	pubsub := s.rdb.Subscribe(ctx, s.channel)
	ch := pubsub.Channel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-ch: // в любом сообщении перезагружаем весь список подсетей
			if !ok {
				return fmt.Errorf("redis pubsub channel closed")
			}

			if err := s.subnetHolder.ReloadSubnets(ctx); err != nil {
				return fmt.Errorf("reload subnet policy: %w", err)
			}
		}
	}
}
