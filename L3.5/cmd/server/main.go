package main

import (
	"context"
	"eventbooker/internal/config"
	"eventbooker/internal/handlers"
	"eventbooker/internal/repository/postgres"
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
	defer dbPool.Master.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dbPool.Master.PingContext(ctx); err != nil {
		log.Error("database is unreachable", slog.Any("err", err))
		os.Exit(1)
	}
	log.Info("successfully connected to database")

	// repositories & handlers
	eventRepo := postgres.NewPostgresRepository(dbPool)
	eventHandler := handlers.NewEventHandler(eventRepo, log)

	// background workers
	eventHandler.SetupScheduler()
	log.Info("background scheduler started")

	// routing
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/events", func(r chi.Router) {
		r.Get("/", eventHandler.ListEvents)
		r.Post("/", eventHandler.CreateEvent)
		r.Get("/{id}", eventHandler.GetEvent)
		r.Post("/{id}/book", eventHandler.BookSpace)
		r.Post("/{id}/confirm", eventHandler.ConfirmBooking)
	})

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
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	if err := srv.Shutdown(pingCtx); err != nil {
		log.Error("Server forced to shutdown", slog.Any("err", err))
	}
	log.Info("Server exiting")
}

const (
	localEnv = "local"
	prodEnv  = "prod"
	devEnv   = "dev"
)

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
