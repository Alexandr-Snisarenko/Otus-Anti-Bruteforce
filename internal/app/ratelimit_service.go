package app

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain/ratelimit"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain/subnetlist"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
)

type RateLimiterUseCase interface {
	Check(ctx context.Context, login, password string, ip net.IP) (bool, error)
	Reset(ctx context.Context, login, password string, ip net.IP) error
}

// Проверка реализации интерфейса RateLimiterUseCase на этапе компиляции.
var (
	_ RateLimiterUseCase = (*RateLimiterService)(nil)
)

type RateLimiterService struct {
	subnetPolicer ports.SubnetPolicer
	limitChecker  ports.LimitChecker
}

func NewRateLimiterService(
	subnetRepo ports.SubnetRepo,
	limitRepo ports.LimiterRepo,
	cfg *config.Limits,
) (*RateLimiterService, error) {
	///// SubnetPolicer /////
	// Для проверки черных/белых списков IP адресов
	// используем inmemory объект subnetPolicer на основе subnetlist.SubnetPolicer
	// с загрузкой данных из subnetRepo при старте и обновлением по требованию.
	subnetPolicer, err := subnetlist.NewSubnetPolicer(subnetRepo)
	if err != nil {
		return nil, fmt.Errorf("subnet policy init: %w", err)
	}

	///// LimitChecker /////
	// Для проверки и учёта попыток аутентификации
	// используем rate limit сервис на основе ratelimit.LimitChecker
	// с лимитами из конфига и проверкой в limitRepo
	limits := subnetlist.LimitsFromConfig(cfg)
	limitChecker := ratelimit.NewLimitChecker(limitRepo, limits)

	return &RateLimiterService{
		subnetPolicer: subnetPolicer,
		limitChecker:  limitChecker,
	}, nil
}

func (s *RateLimiterService) Init(ctx context.Context) error {
	// Первоначальная загрузка списков подсетей
	if err := s.subnetPolicer.ReloadSubnets(ctx); err != nil {
		return fmt.Errorf("initial subnet lists load: %w", err)
	}
	return nil
}

func (s *RateLimiterService) Check(ctx context.Context, login, password string, ip net.IP) (bool, error) {
	// Проверка whitelist/blacklist
	decision := s.subnetPolicer.Check(ip)
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
	// Считаем, что пустые логин/пароль/IP - валидные значения для проверки лимитов.
	// В ТЗ не оговорено иное.
	// Если нужно игнорировать или не допускать пустые значения, то здесь надо добавить проверки.
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

// Reset сбрасывает счетчики лимитов для переданных параметров.
// Почему то в ТЗ нет требования сброса пароля, но логично его добавить.
func (s *RateLimiterService) Reset(ctx context.Context, login, password string, ip net.IP) error {
	passwordHash := hashPassword(password)

	if err := s.limitChecker.Reset(ctx, domain.LoginLimit, login); err != nil {
		return err
	}
	if err := s.limitChecker.Reset(ctx, domain.PasswordLimit, passwordHash); err != nil {
		return err
	}
	if err := s.limitChecker.Reset(ctx, domain.IPLimit, ip.String()); err != nil {
		return err
	}
	return nil
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
}

func (s *RateLimiterService) ReloadSubnets(ctx context.Context) error {
	return s.subnetPolicer.ReloadSubnets(ctx)
}
