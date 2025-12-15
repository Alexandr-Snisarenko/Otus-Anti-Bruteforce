package ratelimit

import (
	"context"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
)

var _ ports.LimitChecker = (*LimitChecker)(nil)

type LimitChecker struct {
	limiters map[domain.LimitType]*Limiter
}

type Limits map[domain.LimitType]Rule

func NewLimitChecker(storage ports.LimiterRepo, limits Limits) *LimitChecker {
	limiters := make(map[domain.LimitType]*Limiter)
	for lt, rule := range limits {
		limiters[lt] = NewLimiter(storage, string(lt), rule)
	}
	return &LimitChecker{
		limiters: limiters,
	}
}

func (lc *LimitChecker) Allow(ctx context.Context, lt domain.LimitType, key string) (bool, error) {
	limiter, exists := lc.limiters[lt]
	if !exists {
		return false, nil
	}
	return limiter.Allow(ctx, key)
}

func (lc *LimitChecker) Reset(ctx context.Context, lt domain.LimitType, key string) error {
	limiter, exists := lc.limiters[lt]
	if !exists {
		return nil
	}
	return limiter.Reset(ctx, key)
}
