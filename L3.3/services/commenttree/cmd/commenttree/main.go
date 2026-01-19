package main

import (
	"commenttree/internal/config"
	"commenttree/internal/handlers"
	"commenttree/internal/middleware/logger"
	"commenttree/internal/store/postgres"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.MustLoad("config/local.yaml")
	log := setupLogger(cfg.Env)

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Storage.Host,
		cfg.Storage.Port,
		cfg.Storage.User,
		cfg.Storage.Password,
		cfg.Storage.DBName,
		cfg.Storage.SSLMode,
	)

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Error("failed to open postgres", slog.Any("err", err))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Error("failed to ping postgres", slog.Any("err", err))
		os.Exit(1)
	}

	pgStore := postgres.New(sqlDB)
	hl := handlers.New(pgStore, pgStore, pgStore, pgStore, pgStore)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(logger.New(log))
	r.Use(middleware.Recoverer)

	// routes
	r.Get("/comments", hl.GetComments)
	r.Post("/comments", hl.CreateComment)
	r.Delete("/comments/{id}", hl.DeleteComment)
	r.Get("/roots", hl.GetRootCommments)
	r.Get("/search", hl.SearchComments)

	// web
	fs := http.FileServer(http.Dir("./web"))
	r.Get("/*", fs.ServeHTTP)

	addr := cfg.HTTPServer.Address
	log.Info("server starting", slog.String("addr", addr))

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server stopped with error", slog.Any("err", err))
		os.Exit(1)
	}
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
