package redis

import (
	"context"
	"time"

	redisclient "github.com/wb-go/wbf/redis"
)

type Cache struct {
	client *redisclient.Client
}

func New(client *redisclient.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key)
}

func (c *Cache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if ttl > 0 {
		return c.client.SetWithExpiration(ctx, key, value, ttl)
	}
	return c.client.Set(ctx, key, value)
}

func (c *Cache) Del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key)
}
