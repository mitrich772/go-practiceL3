package service

import (
	"context"
	"log/slog"
	"shortener/internal/cache"
	"shortener/internal/store"
)

type Redirector interface {
	Resolve(ctx context.Context, alias string) (string, error)
}

type RedirectService struct {
	store store.Store
	cache cache.Cache
	log   *slog.Logger
}

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

func (s *RedirectService) Resolve(ctx context.Context, alias string) (string, error) {
	// 1. cache hit
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

	// 2. cache miss → store
	originalURL, err := s.store.GetURL(ctx, alias)
	if err != nil {
		return "", ErrShortNotFound
	}

	// 3. write-through cache
	if err := s.cache.Set(ctx, alias, originalURL, cacheTTL); err != nil {
		s.log.Warn("failed to set cache",
			slog.String("short", alias),
			slog.Any("error", err),
		)
	}

	return originalURL, nil
}
