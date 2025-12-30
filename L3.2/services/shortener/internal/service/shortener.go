package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"shortener/internal/cache"
	"shortener/internal/store"

	"github.com/lib/pq"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type Shortener interface {
	Shorten(ctx context.Context, url, alias string) (string, error)
}

const cacheTTL = 10 * time.Minute

type ShortenerService struct {
	store store.Store
	cache cache.Cache
	log   *slog.Logger
}

func NewShortener(
	store store.Store,
	cache cache.Cache,
	log *slog.Logger,
) *ShortenerService {
	return &ShortenerService{
		store: store,
		cache: cache,
		log:   log.With(slog.String("service", "shortener")),
	}
}

func (s *ShortenerService) Shorten(ctx context.Context, url, alias string) (string, error) {
	if alias != "" {
		return s.withUserAlias(ctx, alias, url)
	}
	return s.withGeneratedAlias(ctx, url)
}

func (s *ShortenerService) withUserAlias(ctx context.Context, alias, url string) (string, error) {
	if err := s.store.SaveURL(ctx, alias, url); err != nil {
		if isUniqueViolation(err) {
			return "", ErrAliasAlreadyExists
		}
		return "", err
	}

	if err := s.cache.Set(ctx, alias, url, cacheTTL); err != nil {
		s.log.Warn("failed to set cache",
			slog.String("short", alias),
			slog.Any("error", err),
		)
	}

	return alias, nil
}

func (s *ShortenerService) withGeneratedAlias(ctx context.Context, url string) (string, error) {
	const (
		aliasLen    = 8
		maxAttempts = 5
	)

	for i := 0; i < maxAttempts; i++ {
		alias, err := gonanoid.New(aliasLen)
		if err != nil {
			return "", err
		}

		if err = s.store.SaveURL(ctx, alias, url); err == nil {
			if err := s.cache.Set(ctx, alias, url, cacheTTL); err != nil {
				s.log.Warn("failed to set cache",
					slog.String("short", alias),
					slog.Any("error", err),
				)
			}
			return alias, nil
		}

		if isUniqueViolation(err) {
			s.log.Debug("alias collision",
				slog.String("short", alias),
			)
			continue
		}

		return "", err
	}

	s.log.Error("failed to generate unique alias",
		slog.Int("attempts", maxAttempts),
	)

	return "", ErrFailedToGenerateAlias
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}
