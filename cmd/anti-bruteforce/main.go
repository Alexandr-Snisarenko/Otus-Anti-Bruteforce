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

	pbv1 "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/api/proto/anti_bruteforce/v1"
	sub "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/adapters/redissubscriber"
	pub "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/adapters/subnetupdatepublisher"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/app"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/config"
	grpcserver "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/delivery/grpc"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/delivery/grpc/interceptors"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/factory"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/logger"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ports"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/memory"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/postgresdb"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/storage/redisdb"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/version"
	"github.com/redis/go-redis/v9"
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
		subnetRepo         ports.SubnetRepo
		limitRepo          ports.LimiterRepo
		redisPolicerClient *redis.Client
		subscriber         *sub.SubnetUpdatesSubscriber
		publisher          ports.SubnetUpdatesPublisher
		logg               *logger.Logger
		err                error
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
		subnetRepo, err = postgresdb.NewSubnetListDB(cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to initialize PostgreSQL subnet repository: %w", err)
		}
	}
	// Репозиторий для хранения данных по rate limiting
	if localWorcmode {
		limitRepo = memory.NewBucketsDB() // In-memory storage for rate limiting
	} else {
		redisPolicerClient, err = factory.NewClientPolicer(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to initialize Redis client for rate limit repository: %w", err)
		}

		limitRepo = redisdb.NewBucketsRepo(redisPolicerClient)
	}

	// Контекст для управления горутинами и подписчиком с сигналами ОС
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	///////////////////////////////////////////////////////////////////////
	// --------- Инициализация сервисов ---------
	//----- Основной сервис анти-брутфорс защиты rateLimiterSvc ---------
	rateLimiterSvc, err := app.NewRateLimiterService(subnetRepo, limitRepo, &cfg.Limits)
	if err != nil {
		return fmt.Errorf("failed to initialize rate limiter service: %w", err)
	}
	// Инициализация сервиса (загрузка списков подсетей)
	if err := rateLimiterSvc.Init(rootCtx); err != nil {
		return fmt.Errorf("rate limiter service init: %w", err)
	}
	// Redis Subscriber для обновления списков подсетей
	// Подписчик на обновления списков подсетей из Redis Pub/Sub
	// запускаем только в случае работы не в локальном режиме
	// запускается в отдельной горутине ниже
	if !localWorcmode {
		// Отдельный Redis клиент для подписчика
		rdc, err := factory.NewClientSubscriber(&cfg.Database)
		if err != nil {
			return fmt.Errorf("failed to initialize Redis client for subscriber: %w", err)
		}
		subscriber = sub.NewSubnetUpdatesSubscriber(rdc, rateLimiterSvc, cfg.Database.Redis.Subscriber.SubnetsChannel)
	}

	// ---- Сервис для управления черными/белыми списками подсетей в БД subnetRepo ----

	// Publisher для публикации обновлений списков подсетей
	if localWorcmode {
		// В локальном режиме - publisher сразу обновляет списки сетей в subnetHolder (rateLimiterSvc)
		publisher = pub.NewLocalSubnetUpdatesPublisher(rateLimiterSvc)
	} else {
		if redisPolicerClient == nil {
			return errors.New("redis client for publisher is nil")
		}
		// Внешний режим - publisher шлёт уведомления в Redis Pub/Sub канал
		publisher, err = pub.NewRedisSubnetUpdatesPublisher(redisPolicerClient, cfg.Database.Redis.Subscriber.SubnetsChannel)
		if err != nil {
			return fmt.Errorf("failed to initialize Redis subnet updates publisher: %w", err)
		}
	}

	// Сервис управления черными/белыми списками подсетей
	subnetListSvc := app.NewSubnetListService(subnetRepo, publisher)

	///////////////////////////////////////////////////////////////////////
	// -------- gRPC-сервер --------
	addr := net.JoinHostPort(cfg.Server.Address, fmt.Sprint(cfg.Server.Port))
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	grpcSrv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRequestIDInterceptor(),
			interceptors.UnaryLoggingInterceptor(logg),
		),
	)

	// регистрируем сервисы в gRPC-сервере
	srv := grpcserver.NewServer(rateLimiterSvc, subnetListSvc)
	pbv1.RegisterAntiBruteforceServer(grpcSrv, srv)

	// -------- Запуск gRPC-сервера и подписчика в горутинах --------
	// Используем errgroup для управления горутинами и обработки ошибок
	g, ctx := errgroup.WithContext(rootCtx)

	// ------- Graceful shutdown --------
	// Горутина слушает ctx.Done и делает graceful shutdown gRPC сервера
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
