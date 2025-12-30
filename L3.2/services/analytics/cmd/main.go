package main

import (
	"analytics/internal/config"
	"analytics/internal/handlers"
	mwLogeer "analytics/internal/middleware/logger"
	"analytics/internal/service"
	"analytics/internal/store/postgres"

	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// TODO: rewtrite for analytics
func main() {
	// config
	cfg := config.MustLoad("config/local.yaml")

	// logger
	log := setupLogger(cfg.Env)

	// store : postgres
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Storage.Host,
		cfg.Storage.Port,
		cfg.Storage.User,
		cfg.Storage.Password,
		cfg.Storage.DBName,
		cfg.Storage.SSLMode,
	)

	log.Info(
		"postgres connected",
		slog.String("host", cfg.Storage.Host),
		slog.Int("port", cfg.Storage.Port),
		slog.String("db", cfg.Storage.DBName),
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error("failed to open postgres connection",
			slog.String("op", "postgres.open"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	sqlDB.SetMaxOpenConns(cfg.Storage.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Storage.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Storage.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		log.Error("failed to ping postgres",
			slog.String("op", "postgres.ping"),
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	store := postgres.New(sqlDB)

	// service init
	al := service.New(store)

	// router : chi
	hl := handlers.New(log, al)
	r := chi.NewRouter()

	// standart chi middleware work with slog ?
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mwLogeer.New(log))
	r.Use(middleware.Recoverer)

	r.Post("/events/click", hl.Click)
	r.Get("/analytics/{short}", hl.GetAnalytics)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	log.Info(
		"HTTP server listening",
		slog.String("addr", cfg.HTTPServer.Address),
	)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error(
			"http server stopped",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	c := make(chan os.Signal, 1)
	<-c
}

const (
	localEnv = "local"
	prodEnv  = "prod"
	devEnv   = "dev"
)

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger
	switch env {
	case localEnv:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	// TODO: prod, dev ?
	return log
}
