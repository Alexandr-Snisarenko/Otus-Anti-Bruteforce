package ports

import (
	"context"

	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
)

// LimitChecker — интерфейс для проверки и сброса лимитов.
type LimitChecker interface {
	Allow(ctx context.Context, lt domain.LimitType, key string) (bool, error)
	Reset(ctx context.Context, lt domain.LimitType, key string) error
}
