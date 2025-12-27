package factory

import (
	"context"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/config"
	"github.com/redis/go-redis/v9"
)

func NewClientPolicer(cfg *config.Database) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Address,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  cfg.Redis.Policer.DialTimeout,
		ReadTimeout:  cfg.Redis.Policer.ReadTimeout,
		WriteTimeout: cfg.Redis.Policer.WriteTimeout,
		PoolSize:     cfg.Redis.Policer.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	return rdb, nil
}

func NewClientSubscriber(cfg *config.Database) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:        cfg.Redis.Address,
		Password:    cfg.Redis.Password,
		DB:          cfg.Redis.DB,
		ReadTimeout: cfg.Redis.Subscriber.ReadTimeout,
		PoolSize:    cfg.Redis.Subscriber.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	return rdb, nil
}
