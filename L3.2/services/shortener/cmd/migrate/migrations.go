package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"shortener/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {

	action := flag.String("action", "up", "up | down | step")
	steps := flag.Int("n", 1, "Количество шагов для step")
	flag.Parse()

	cfg := config.MustLoad("config/local.yaml")

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Storage.User,
		cfg.Storage.Password,
		cfg.Storage.Host,
		cfg.Storage.Port,
		cfg.Storage.DBName,
		cfg.Storage.SSLMode,
	)

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatalf("Ошибка создания мигратора: %v", err)
	}

	switch *action {
	case "up":
		err = m.Up()
	case "down":
		err = m.Down()
	case "step":
		err = m.Steps(*steps)
	default:
		log.Fatalf("Неизвестное действие: %s", *action)
	}

	if err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	log.Println("Миграция завершена")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
