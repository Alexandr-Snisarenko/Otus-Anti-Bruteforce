package subnetupdatepublisher

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisSubnetUpdatesPublisher struct {
	rdb     *redis.Client
	channel string
}

func (p *RedisSubnetUpdatesPublisher) PublishSubnetUpdated(ctx context.Context) error {
	return p.rdb.Publish(ctx, p.channel, "reload").Err()
}

func NewRedisSubnetUpdatesPublisher(
	rdb *redis.Client,
	channel string,
) (*RedisSubnetUpdatesPublisher, error) {
	return &RedisSubnetUpdatesPublisher{rdb: rdb, channel: channel}, nil
}
