// Package main — entrypoint HTTP-сервиса salestracker.
//
// Каноничный порядок инициализации (как в L3.5):
// config → logger → DSN + dbpg.New → ping → repo → handlers → router → server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/wb-go/wbf/dbpg"

	"salestracker/internal/config"
	"salestracker/internal/handlers"
	customLog "salestracker/internal/middleware/logger"
	"salestracker/internal/repo/postgres"
)

const (
	localEnv = "local"
	prodEnv  = "prod"
	devEnv   = "dev"
)

func main() {
	// config & logger
	cfg := config.MustLoad("config/local.yaml")
	log := setupLogger(cfg.Env)
	log.Info("config loaded", slog.String("env", cfg.Env))

	// db init
	masterDSN := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName, cfg.DB.SSLMode)

	dbOpts := &dbpg.Options{
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
	}

	dbPool, err := dbpg.New(masterDSN, nil, dbOpts)
	if err != nil {
		log.Error("failed to create db pool", slog.Any("err", err))
		os.Exit(1)
	}
	defer dbPool.Master.Close() //nolint:errcheck

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := dbPool.Master.PingContext(pingCtx); err != nil {
		log.Error("database is unreachable", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("successfully connected to database")

	// repositories & handlers
	itemRepo := postgres.New(dbPool)
	h := handlers.New(itemRepo, itemRepo, itemRepo, itemRepo, itemRepo, itemRepo, log)

	// routing
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customLog.New(log))
	r.Use(middleware.Recoverer)

	r.Post("/items", h.CreateItem)
	r.Get("/items", h.ListItems)
	r.Get("/items.csv", h.ExportCSV)
	r.Get("/items/{id}", h.GetItem)
	r.Put("/items/{id}", h.UpdateItem)
	r.Delete("/items/{id}", h.DeleteItem)

	r.Get("/analytics", h.GetAnalytics)

	// static files
	fs := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/index.html", http.StatusFound)
	})

	// server start
	srv := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: r,
	}

	go func() {
		log.Info("Starting HTTP server", slog.String("addr", cfg.Server.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed to start server", slog.Any("err", err))
			os.Exit(1)
		}
	}()
	log.Info("server is ready to handle requests")

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Shutting down server...")

	shCtx, shCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shCancel()
	if err := srv.Shutdown(shCtx); err != nil {
		log.Error("Server forced to shutdown", slog.Any("err", err))
	}
	log.Info("Server exiting")
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
