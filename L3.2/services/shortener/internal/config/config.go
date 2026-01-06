// Package config содержит структуры конфигурации и загрузчик конфигурации.
package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config описывает конфигурацию shortener-сервиса.
type Config struct {
	Env       string `yaml:"env" env-default:"local"`
	CachePath string `yaml:"cache_path" env-required:"true"`

	Analytics  Analytics  `yaml:"analytics"`
	Storage    Storage    `yaml:"storage"`
	HTTPServer HTTPServer `yaml:"http_server"`
}

// Analytics содержит настройки подключения к сервису аналитики.
type Analytics struct {
	URL string `yaml:"url" env-required:"true"`
}

// Storage содержит настройки подключения к PostgreSQL.
type Storage struct {
	Host            string        `yaml:"host" env-required:"true"`
	Port            int           `yaml:"port" env-default:"5432"`
	User            string        `yaml:"user" env-required:"true"`
	Password        string        `yaml:"password" env-required:"true"`
	DBName          string        `yaml:"dbname" env-required:"true"`
	SSLMode         string        `yaml:"sslmode" env-default:"disable"`
	MaxOpenConns    int           `yaml:"max_open_conns" env-default:"10"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env-default:"5"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env-default:"30m"`
}

// HTTPServer содержит настройки HTTP-сервера.
type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

// MustLoad загружает конфиг из файла по пути configPath.
// Если configPath пустой, используется переменная окружения CONFIG_PATH.
// При любой ошибке завершает процесс через log.Fatal/log.Fatalf.
func MustLoad(configPath string) *Config {
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}

	if configPath == "" {
		log.Fatal("config path not provided: pass argument or set CONFIG_PATH")
	}

	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
