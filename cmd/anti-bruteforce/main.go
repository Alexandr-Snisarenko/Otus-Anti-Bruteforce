package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/api/proto/anti_bruteforce/v1"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/adapters/redissubscriber"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/app"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	grpcserver "github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/delivery/grpc"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/logger"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ports"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/storage/memory"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/storage/postgresql"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/storage/redisclient"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/version"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "/etc/anti-bruteforce/config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	if flag.Arg(0) == "version" {
		version.PrintVersion()
		return
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load confige: %v\n", err)
		os.Exit(1)
	}

	// Запуск основного приложения
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "anti-bruteforce exited with error: %v\n", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	var (
		subnetRepo ports.SubnetRepo
		limitRepo  ports.LimiterRepo
		subscriber *redissubscriber.SubnetUpdatesSubscriber
		logg       *logger.Logger
		err        error
	)

	// --------- Инициализация логгера ---------
	if cfg.Logger.File != "" {
		// Инициализация логгера с выводом в файл
		f, err := os.OpenFile(cfg.Logger.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			// логер не инициализирован, пишем в stderr
			return fmt.Errorf("failed to open log file: %w", err)
		}
		logg = logger.NewWithWriter(f, &cfg.Logger)
		defer f.Close()
	} else {
		// Инициализация логгера с выводом в stdout
		logg = logger.New(&cfg.Logger)
		logg.Info("Logger initialized", "level", cfg.Logger.Level)
	}

	// --------- Инициализация репозиториев ---------
	localWorcmode := cfg.Database.Workmode == "local"
	// Репозиторий списка подсетей.
	if localWorcmode {
		subnetRepo = memory.NewSubnetListDB() // In-memory storage for subnet lists
	} else {
		subnetRepo, err = postgresql.New(cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to initialize PostgreSQL subnet repository: %w", err)
		}
	}
	// Репозиторий для хранения данных по rate limiting
	if localWorcmode {
		limitRepo = memory.NewBucketsDB() // In-memory storage for rate limiting
	} else {
		limitRepo, err = redisclient.New(cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to initialize Redis rate limit repository: %w", err)
		}
	}

	// Контекст для управления горутинами и подписчиком с сигналами ОС
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --------- Инициализация сервисов ---------
	// Основной сервис анти-брутфорс защиты
	rateLimiterSvc, err := app.NewRateLimiterService(subnetRepo, limitRepo, &cfg.Limits)
	if err != nil {
		return fmt.Errorf("failed to initialize rate limiter service: %w", err)
	}
	// Инициализация сервиса (загрузка списков подсетей)
	if err := rateLimiterSvc.Init(rootCtx); err != nil {
		return fmt.Errorf("rate limiter service init: %w", err)
	}

	// Сервис для управления черными/белыми списками подсетей в БД subnetRepo
	subnetListSvc := app.NewSubnetListService(subnetRepo)

	// -------- gRPC-сервер --------
	addr := net.JoinHostPort(cfg.Server.Address, fmt.Sprint(cfg.Server.Port))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	grpcSrv := grpc.NewServer(
	// сюда потом добавить interceptors: логирование, метаданные и т.п.
	)

	// регистрируем сервисы в gRPC-сервере
	srv := grpcserver.NewServer(rateLimiterSvc, subnetListSvc)
	pbv1.RegisterAntiBruteforceServer(grpcSrv, srv)

	// -------- Redis Subscriber для обновления списков подсетей --------
	// Подписчик на обновления списков подсетей из Redis Pub/Sub
	// запускаем только в случае работы не в локальном режиме
	if !localWorcmode {
		subscriber, err = redissubscriber.NewSubnetUpdatesSubscriber(&cfg.Database, rateLimiterSvc)
		if err != nil {
			return fmt.Errorf("failed to initialize subnet updates subscriber: %w", err)
		}
	}

	// -------- Запуск gRPC-сервера и подписчика в горутинах --------
	// Используем errgroup для управления горутинами и обработки ошибок
	g, ctx := errgroup.WithContext(rootCtx)

	// Горутина, которая слушает ctx.Done и делает graceful shutdown gRPC сервера
	g.Go(func() error {
		<-ctx.Done() // ждём отмены контекста (сигнал или падение другой горутины)

		logg.Info("shutting down gRPC server...")
		done := make(chan struct{})
		go func() {
			grpcSrv.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			logg.Info("gRPC server stopped gracefully")
		case <-time.After(5 * time.Second):
			logg.Info("gRPC server force stop")
			grpcSrv.Stop()
		}

		return ctx.Err()
	})

	// Запускаем подписчика на обновления списков подсетей
	if subscriber != nil {
		g.Go(func() error {
			logg.Info("starting subnet updates subscriber...")
			return subscriber.Start(ctx)
		})
	}

	// Стартуем сервер в отдельной горутине
	g.Go(func() error {
		logg.Info("gRPC server listening on ", "port", addr)
		return grpcSrv.Serve(lis)
	})

	// Ждём завершения горутин
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logg.Error("error from goroutines", "error", err)
		return err
	}

	logg.Info("application stopped gracefully")
	return nil
}
