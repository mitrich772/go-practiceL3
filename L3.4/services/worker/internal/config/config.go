// Package config содержит структуры конфигурации и загрузчик конфигурации.
package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config описывает конфигурацию сервиса.
type Config struct {
	Env string `yaml:"env" env-default:"local"`

	Server  Server  `yaml:"server"`
	DB      DB      `yaml:"db"`
	Kafka   Kafka   `yaml:"kafka"`
	Storage Storage `yaml:"storage"`
}

// Server содержит настройки HTTP сервера.
type Server struct {
	Addr string `yaml:"addr" env-default:":8080"`
}

// DB содержит настройки подключения к БД.
type DB struct {
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

// Kafka содержит настройки Kafka (для API нам нужен producer).
type Kafka struct {
	Brokers       []string `yaml:"brokers" env-required:"true"`
	TopicImagesIn string   `yaml:"topic_images_in" env-default:"images.in"`

	Consumer KafkaConsumer `yaml:"consumer"`
}

type KafkaConsumer struct {
	GroupID string `yaml:"group_id" env-default:"image-worker"`
}

// Storage содержит пути для хранения изображений.
type Storage struct {
	OriginalDir  string `yaml:"original_dir" env-default:"./data/original"`
	ProcessedDir string `yaml:"processed_dir" env-default:"./data/processed"`
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
