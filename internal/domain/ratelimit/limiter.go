package ratelimit

import (
	"context"
	"time"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
)

// Limiter - реализация лимитера по алгоритму скользящего окна.
// Для каждого типа лимита (limitType) отдельный экземпляр Limiter.
// Пример типов лимитов: login, password, ip.
// Пример использования:
//		loginLimiter := ratelimit.NewLimiter(storage, "login", ratelimit.Rule{Limit: 5, Window: time.Minute})
//		ipLimiter := ratelimit.NewLimiter(storage, "ip", ratelimit.Rule{Limit: 20, Window: time.Minute})

// Rule описывает правило для одного типа лимита (например, login или ip).
// Пример: Rule{Limit: 10, Window: time.Minute}.
type Rule struct {
	Limit  int
	Window time.Duration
}

// Limiter — реализация по алгоритму скользящего окна.
// Для каждого типа лимита (limitType) отдельный экземпляр Limiter.
type Limiter struct {
	storage   ports.LimiterRepo
	limitType string // тип лимита (login, password или ip)
	rule      Rule
}

// NewLimiter создает новый экземпляр Limiter для заданного типа лимита и правила.
func NewLimiter(st ports.LimiterRepo, limitType string, r Rule) *Limiter {
	return &Limiter{
		storage:   st,
		limitType: limitType,
		rule:      r,
	}
}

// Allow проверяет, разрешена ли по лимитеру попытка для данного ключа.
func (l *Limiter) Allow(ctx context.Context, key string) (bool, error) {
	key = l.limitType + ":" + key
	return l.storage.Allow(ctx, key, l.rule.Limit, l.rule.Window)
}

// Reset сбрасывает счетчик попыток для данного лимитера для данного ключа.
func (l *Limiter) Reset(ctx context.Context, key string) error {
	key = l.limitType + ":" + key
	return l.storage.Reset(ctx, key)
}
