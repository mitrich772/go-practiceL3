package main

import (
	"errors"
	"flag"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"eventbooker/internal/config"
)

func main() {

	action := flag.String("action", "up", "up | down | step")
	steps := flag.Int("n", 1, "Количество шагов для step")
	flag.Parse()

	cfg := config.MustLoad("config/local.yaml")

	dbURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.DBName,
		cfg.DB.SSLMode,
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

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	log.Println("Миграция завершена")
}
