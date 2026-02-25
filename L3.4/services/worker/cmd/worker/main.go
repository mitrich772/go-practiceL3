package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"worker/internal/config"
	"worker/internal/kafka_consumer"

	pgrepo "worker/internal/repo/postgres"
	localstorage "worker/internal/storage"

	"github.com/wb-go/wbf/dbpg"
	wbfkafka "github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"
)

const (
	localEnv = "local"
	prodEnv  = "prod"
	devEnv   = "dev"
)

func main() {
	cfg := config.MustLoad("config/worker_local.yaml")
	log := setupLogger(cfg.Env)

	log.Info("worker config loaded",
		slog.String("env", cfg.Env),
		slog.Any("brokers", cfg.Kafka.Brokers),
		slog.String("topic", cfg.Kafka.TopicImagesIn),
		slog.String("group_id", cfg.Kafka.Consumer.GroupID),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// --- storage ---
	if err := os.MkdirAll(cfg.Storage.OriginalDir, 0o755); err != nil {
		log.Error("failed to create original dir", slog.Any("err", err))
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.Storage.ProcessedDir, 0o755); err != nil {
		log.Error("failed to create processed dir", slog.Any("err", err))
		os.Exit(1)
	}

	fs := localstorage.NewFileStorage(cfg.Storage.OriginalDir, cfg.Storage.ProcessedDir)
	log.Info("storage ready",
		slog.String("original_dir", cfg.Storage.OriginalDir),
		slog.String("processed_dir", cfg.Storage.ProcessedDir),
	)

	// --- postgres repo ---
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.DBName,
		cfg.DB.SSLMode,
	)

	opts := &dbpg.Options{
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
	}

	pg, err := dbpg.New(dsn, nil, opts) // nil => без slave
	if err != nil {
		log.Error("failed to init dbpg", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() { _ = pg.Master.Close() }()

	ctxPing, cancelPing := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancelPing()
	if err := pg.Master.PingContext(ctxPing); err != nil {
		log.Error("failed to ping postgres", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("postgres connected")

	dbRepo := pgrepo.New(pg, log)

	// --- kafka consumer ---
	consumer := wbfkafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.TopicImagesIn, cfg.Kafka.Consumer.GroupID)
	defer func() { _ = consumer.Close() }()

	fetchRetry := retry.Strategy{Attempts: 10, Delay: 1 * time.Second, Backoff: 2}

	workerConsumer := kafka_consumer.New(log, consumer, fetchRetry, fs, dbRepo)

	// ВАЖНО: StartConsume блокирует — запускаем в goroutine
	go workerConsumer.StartConsume(ctx)

	<-stop
	log.Info("shutdown signal received")
	cancel()

	time.Sleep(300 * time.Millisecond)
	log.Info("worker shutdown complete")
}

func setupLogger(env string) *slog.Logger {
	switch env {
	case localEnv:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case devEnv:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case prodEnv:
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	default:
		return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
}
