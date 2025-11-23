package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain/ratelimit"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain/subnetlist"
)

type RateLimiterService struct {
	listPolicy   *subnetlist.SubnetPolicy
	limitChecker *ratelimit.LimitChecker
}

func NewRateLimiterService(
	policy *subnetlist.SubnetPolicy,
	checker *ratelimit.LimitChecker,
) *RateLimiterService {
	return &RateLimiterService{
		listPolicy:   policy,
		limitChecker: checker,
	}
}

func (s *RateLimiterService) Check(ctx context.Context, login, password string, ip net.IP) (bool, error) {
	// Проверка whitelist/blacklist
	decision := s.listPolicy.Check(ip)
	switch decision {
	case domain.DecisionDeny:
		return false, nil
	case domain.DecisionAllow:
		return true, nil
	case domain.DecisionContinue:
		// Продолжаем проверку лимитов
	}

	// Хешируем пароль перед проверкой лимитов
	passwordHash := hashPassword(password)

	// Три проверки: по логину, паролю, IP
	if okLogin, err := s.limitChecker.Allow(ctx, domain.LoginLimit, login); err != nil || !okLogin {
		return false, err
	}
	if okPassword, err := s.limitChecker.Allow(ctx, domain.PasswordLimit, passwordHash); err != nil || !okPassword {
		return false, err
	}
	if okIP, err := s.limitChecker.Allow(ctx, domain.IPLimit, ip.String()); err != nil || !okIP {
		return false, err
	}

	return true, nil
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}
