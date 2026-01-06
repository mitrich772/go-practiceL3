// Package cache содержит интерфейс кеша и связанные типы.
package cache

import (
	"context"
	"time"
)

//go:generate mockgen -source=cache.go -destination=./mocks/mock_cache.go -package=cachemocks

// Cache описывает операции кеша, используемые сервисами приложения.
type Cache interface {
	// Get возвращает значение по ключу или ошибку, если ключ отсутствует/кеш недоступен.
	Get(ctx context.Context, key string) (string, error)

	// Set сохраняет значение по ключу с TTL
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Del удаляет ключ из кеша.
	Del(ctx context.Context, key string) error
}
