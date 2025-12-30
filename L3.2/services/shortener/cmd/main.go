package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"shortener/internal/cache/redis"
	"shortener/internal/config"
	"shortener/internal/handlers"
	mwLogeer "shortener/internal/middleware/logger"
	"shortener/internal/service"
	"shortener/internal/store/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	redisclient "github.com/wb-go/wbf/redis"
)

func main() {
	// config
	cfg := config.MustLoad("config/local.yaml")

	// logger
	log := setupLogger(cfg.Env)

	// cache : reddis
	redisClient := redisclient.New(cfg.CachePath, "", 0)
	cache := redis.New(redisClient)

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
	sh := service.NewShortener(store, cache, log)
	rd := service.NewRedirect(store, cache, log)

	// router : chi
	hl := handlers.New(log, sh, rd, cfg.Analytics.URL)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mwLogeer.New(log))
	r.Use(middleware.Recoverer)

	// api
	r.Post("/shorten", hl.Shorten)
	r.Get("/s/{short_url}", hl.Redirect)
	r.Get("/analytics/{short_url}", hl.ProxyAnalytics)

	// static
	workDir, _ := os.Getwd()
	staticDir := filepath.Join(workDir, "web", "static")

	fs := http.FileServer(http.Dir(staticDir))
	r.Handle("/*", fs)

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	uiURL := fmt.Sprintf("http://%s/", cfg.HTTPServer.Address)

	log.Info(
		"UI available",
		slog.String("url", uiURL),
		slog.String("static_dir", staticDir),
	)

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
