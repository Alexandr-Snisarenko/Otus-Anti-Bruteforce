package redissubscriber

import (
	"context"
	"fmt"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
	"github.com/redis/go-redis/v9"
)

type SubnetUpdatesSubscriber struct {
	rdb          *redis.Client
	channel      string
	subnetHolder ports.SubnetHolder
}

func NewSubnetUpdatesSubscriber(
	cfg *config.Database,
	subnetHolder ports.SubnetHolder,
) (*SubnetUpdatesSubscriber, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Проверяем подключение
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	channel := cfg.Redis.Subscriber.SubnetsChannel

	return &SubnetUpdatesSubscriber{rdb: rdb, channel: channel, subnetHolder: subnetHolder}, nil
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
