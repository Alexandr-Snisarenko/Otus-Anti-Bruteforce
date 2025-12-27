package subnetlist

import (
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/domain/ratelimit"
)

func LimitsFromConfig(cfg *config.Limits) ratelimit.Limits {
	return ratelimit.Limits{
		domain.LoginLimit: {
			Limit:  cfg.LoginAttempts,
			Window: cfg.Window,
		},
		domain.PasswordLimit: {
			Limit:  cfg.PasswordAttempts,
			Window: cfg.Window,
		},
		domain.IPLimit: {
			Limit:  cfg.IPAttempts,
			Window: cfg.Window,
		},
	}
}
