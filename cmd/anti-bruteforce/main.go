package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/app"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain/ratelimit"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/domain/subnetlist"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/logger"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/storage/memory"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/version"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/anti-bruteforce/config.yaml", "Path to configuration file")
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "anti-bruteforce exited with error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	if flag.Arg(0) == "version" {
		version.PrintVersion()
		return nil
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("config load: %w", err)
	}

	logg := logger.New(&cfg.Logger)
	log.Printf("Logger initialized with level: %s", cfg.Logger.Level)

	ctx := context.Background()

	subnetRepo := memory.NewSubnetListDB() // In-memory storage for subnet lists
	limitRepo := memory.NewBucketsDB()     // In-memory storage for rate limits

	policy, err := subnetlist.NewSubnetPolicy(ctx, subnetRepo)
	if err != nil {
		return fmt.Errorf("subnet policy init: %w", err)
	}

	limits := subnetlist.LimitsFromConfig(&cfg.Limits)
	limiter := ratelimit.NewLimitChecker(limitRepo, limits)
	service := app.NewRateLimiterService(policy, limiter)

	_ = service // To avoid unused variable error; in real application, use the service

	logg.Info("Application started")

	// Application logic would go here...

	logg.Info("Application stopped")

	return nil
}
