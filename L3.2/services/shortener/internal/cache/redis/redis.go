// Package redis содержит реализацию кеша на Redis.
package redis

import (
	"context"
	"time"

	redisclient "github.com/wb-go/wbf/redis"
)

// Cache реализует кеш поверх redisclient.Client.
type Cache struct {
	client *redisclient.Client
}

// New создаёт Redis Cache на основе redisclient.Client.
func New(client *redisclient.Client) *Cache {
	return &Cache{client: client}
}

// Get возвращает значение по ключу из Redis.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key)
}

// Set записывает значение по ключу в Redis, используя TTL если ttl > 0.
func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if ttl > 0 {
		return c.client.SetWithExpiration(ctx, key, value, ttl)
	}
	return c.client.Set(ctx, key, value)
}

// Del удаляет ключ из Redis.
func (c *Cache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key)
}
