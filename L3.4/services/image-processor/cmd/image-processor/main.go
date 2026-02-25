package main

import (
	"context"
	"errors"
	"fmt"
	"image-processor/internal/config"
	"image-processor/internal/handlers"
	"image-processor/internal/middleware/logger"
	"image-processor/internal/repo/postgres"
	"image-processor/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"
)

const (
	localEnv = "local"
	prodEnv  = "prod"
	devEnv   = "dev"
)

func main() {
	// config + logger
	cfg := config.MustLoad("config/local.yaml")
	log := setupLogger(cfg.Env)

	log.Info("config loaded", slog.String("env", cfg.Env))

	// storage dirs
	if err := os.MkdirAll(cfg.Storage.OriginalDir, 0o755); err != nil {
		log.Error("failed to create original dir", slog.Any("err", err))
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.Storage.ProcessedDir, 0o755); err != nil {
		log.Error("failed to create processed dir", slog.Any("err", err))
		os.Exit(1)
	}
	localStorage := storage.NewFileStorage(cfg.Storage.OriginalDir, cfg.Storage.ProcessedDir)

	log.Info("storage ready",
		slog.String("original_dir", cfg.Storage.OriginalDir),
		slog.String("processed_dir", cfg.Storage.ProcessedDir),
	)
	// pg conn
	masterDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.DBName,
		cfg.DB.SSLMode,
	)
	opts := &dbpg.Options{
		MaxOpenConns:    cfg.DB.MaxOpenConns,    // если есть в конфиге
		MaxIdleConns:    cfg.DB.MaxIdleConns,    // если есть
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime, // если есть
	}

	pg, err := dbpg.New(masterDSN, nil, opts) // nil => без slave
	if err != nil {
		log.Error("failed to init dbpg", slog.Any("err", err))
		os.Exit(1)
	}

	ctxPing, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := pg.Master.PingContext(ctxPing); err != nil {
		log.Error("failed to ping postgres", slog.Any("err", err))
		os.Exit(1)
	}

	log.Info("postgres connected")

	//postgresStore init
	storePg := postgres.New(pg, log)

	log.Info("postgresStore ready")

	// kafka producer
	producer := kafka.NewProducer(cfg.Kafka.Brokers, cfg.Kafka.TopicImagesIn)
	//enable auto-create topics
	producer.Writer.AllowAutoTopicCreation = true

	log.Info("kafka producer ready",
		slog.Any("brokers", cfg.Kafka.Brokers),
		slog.String("topic", cfg.Kafka.TopicImagesIn),
	)

	// retry strategy для Kafka SendWithRetry
	kafkaRetryStrategy := retry.Strategy{
		Attempts: 3,
		Delay:    2 * time.Second,
		Backoff:  2,
	}

	// router
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(logger.New(log))

	h := handlers.New(log, producer, storePg, localStorage, kafkaRetryStrategy, cfg.UploadConf.MaxUploadBytes)
	r.Post("/upload", h.Upload)
	r.Get("/image/{id}", h.GetImage)
	r.Delete("/image/{id}", h.DeleteImage)

	// server
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("http server started", slog.String("addr", cfg.Server.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped with error", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	<-stop
	log.Info("shutdown signal received")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Error("server shutdown error", slog.Any("err", err))
	}

	if err := producer.Close(); err != nil {
		log.Error("kafka producer close error", slog.Any("err", err))
	}

	_ = pg.Master.Close()
	for _, s := range pg.Slaves {
		_ = s.Close()
	}

	log.Info("shutdown complete")
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
