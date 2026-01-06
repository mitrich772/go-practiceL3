package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lib/pq"

	cachemocks "shortener/internal/cache/mocks"
	storemocks "shortener/internal/store/mocks"
)

func TestShortenerService_Shorten_WithUserAlias_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	st.EXPECT().
		SaveURL(gomock.Any(), "myalias", "https://example.com").
		Return(nil)

	ca.EXPECT().
		Set(gomock.Any(), "myalias", "https://example.com", cacheTTL).
		Return(nil)

	svc := NewShortener(st, ca, testLogger())

	gotAlias, err := svc.Shorten(ctx, "https://example.com", "myalias")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if gotAlias != "myalias" {
		t.Fatalf("expected alias %q, got %q", "myalias", gotAlias)
	}
}

func TestShortenerService_Shorten_WithUserAlias_UniqueViolation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	st.EXPECT().
		SaveURL(gomock.Any(), "taken", "https://example.com").
		Return(&pq.Error{Code: "23505"})

	// не должен писать в кэш при ошибке сохранения
	ca.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewShortener(st, ca, testLogger())

	_, err := svc.Shorten(ctx, "https://example.com", "taken")
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if err != ErrAliasAlreadyExists {
		t.Fatalf("expected ErrAliasAlreadyExists, got %v", err)
	}
}

func TestShortenerService_Shorten_WithGeneratedAlias_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	var savedKey string

	st.EXPECT().
		SaveURL(gomock.Any(), gomock.Any(), "https://example.com").
		DoAndReturn(func(_ context.Context, short, original string) error {
			savedKey = short
			return nil
		})

	ca.EXPECT().
		Set(gomock.Any(), gomock.Any(), "https://example.com", cacheTTL).
		DoAndReturn(func(_ context.Context, key, val string, ttl interface{}) error {
			// ttl тут не interface в реальном коде, но gomock.Any() выше не использовали.
			// Поэтому проще сделаем отдельный вариант ниже без этого DoAndReturn.
			return nil
		}).AnyTimes()

	// Лучше строго и проще: ожидание Set с любым key, но с нужным value/ttl:
	// (Если хочешь строго сравнить key == alias — делай DoAndReturn на Set как ниже)
	ctrl2 := ctrl // чтобы не ругалось на неиспользуемость в некоторых IDE

	_ = ctrl2

	// Перепишем Set ожидание нормально, чтобы сравнить key == savedKey:
	ca.EXPECT().
		Set(gomock.Any(), gomock.Any(), "https://example.com", cacheTTL).
		DoAndReturn(func(_ context.Context, key, val string, ttlDuration interface{}) error {
			// ttlDuration на самом деле time.Duration; но нам важно сравнение key.
			if savedKey != "" && key != savedKey {
				t.Fatalf("expected cache.Set key == savedKey, got key=%q savedKey=%q", key, savedKey)
			}
			return nil
		})

	svc := NewShortener(st, ca, testLogger())

	alias, err := svc.Shorten(ctx, "https://example.com", "")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if len(alias) != 8 {
		t.Fatalf("expected generated alias length 8, got %d (%q)", len(alias), alias)
	}
	if savedKey != "" && alias != savedKey {
		t.Fatalf("expected returned alias == savedKey, got alias=%q savedKey=%q", alias, savedKey)
	}
}

func TestShortenerService_Shorten_WithGeneratedAlias_AllCollisions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	st.EXPECT().
		SaveURL(gomock.Any(), gomock.Any(), "https://example.com").
		Return(&pq.Error{Code: "23505"}).
		Times(5)

	ca.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewShortener(st, ca, testLogger())

	_, err := svc.Shorten(ctx, "https://example.com", "")
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if err != ErrFailedToGenerateAlias {
		t.Fatalf("expected ErrFailedToGenerateAlias, got %v", err)
	}
}

func TestShortenerService_Shorten_WithUserAlias_CacheSetErrorIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	st.EXPECT().
		SaveURL(gomock.Any(), "ok", "https://example.com").
		Return(nil)

	ca.EXPECT().
		Set(gomock.Any(), "ok", "https://example.com", cacheTTL).
		Return(errors.New("redis down"))

	svc := NewShortener(st, ca, testLogger())

	alias, err := svc.Shorten(ctx, "https://example.com", "ok")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if alias != "ok" {
		t.Fatalf("expected alias %q, got %q", "ok", alias)
	}
}
