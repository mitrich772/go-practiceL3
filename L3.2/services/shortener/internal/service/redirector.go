// Package service содержит бизнес-логику shortener-сервиса.
package service

import (
	"context"
	"log/slog"

	"shortener/internal/cache"
	"shortener/internal/store"
)

// Redirector описывает возможность резолва alias в оригинальный URL.
type Redirector interface {
	Resolve(ctx context.Context, alias string) (string, error)
}

// RedirectService реализует Redirector через store и cache.
type RedirectService struct {
	store store.Store
	cache cache.Cache
	log   *slog.Logger
}

// NewRedirect создаёт RedirectService.
func NewRedirect(
	store store.Store,
	cache cache.Cache,
	log *slog.Logger,
) *RedirectService {
	return &RedirectService{
		store: store,
		cache: cache,
		log:   log.With(slog.String("service", "redirect")),
	}
}

// Resolve возвращает оригинальный URL по alias, используя кеш (read-through/write-through).
func (s *RedirectService) Resolve(ctx context.Context, alias string) (string, error) {
	// 1) cache hit
	if originalURL, err := s.cache.Get(ctx, alias); err == nil {
		// touch TTL
		if err := s.cache.Set(ctx, alias, originalURL, cacheTTL); err != nil {
			s.log.Warn("failed to touch cache ttl",
				slog.String("short", alias),
				slog.Any("error", err),
			)
		}
		return originalURL, nil
	}

	// 2) cache miss → store
	originalURL, err := s.store.GetURL(ctx, alias)
	if err != nil {
		return "", ErrShortNotFound
	}

	// 3) write-through cache
	if err := s.cache.Set(ctx, alias, originalURL, cacheTTL); err != nil {
		s.log.Warn("failed to set cache",
			slog.String("short", alias),
			slog.Any("error", err),
		)
	}

	return originalURL, nil
}
