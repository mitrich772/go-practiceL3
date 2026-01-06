package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/golang/mock/gomock"

	cachemocks "shortener/internal/cache/mocks"
	storemocks "shortener/internal/store/mocks"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}

func TestRedirectService_Resolve_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	ca.EXPECT().
		Get(gomock.Any(), "abc123").
		Return("https://example.com", nil)

	// touch TTL
	ca.EXPECT().
		Set(gomock.Any(), "abc123", "https://example.com", cacheTTL).
		Return(nil)

	// store не должен дергаться
	st.EXPECT().
		GetURL(gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewRedirect(st, ca, testLogger())

	got, err := svc.Resolve(ctx, "abc123")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if got != "https://example.com" {
		t.Fatalf("expected url %q, got %q", "https://example.com", got)
	}
}

func TestRedirectService_Resolve_CacheHit_TouchTTLErrorIgnored(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	ca.EXPECT().
		Get(gomock.Any(), "abc123").
		Return("https://example.com", nil)

	// ошибка touch TTL должна игнорироваться
	ca.EXPECT().
		Set(gomock.Any(), "abc123", "https://example.com", cacheTTL).
		Return(errors.New("redis down"))

	st.EXPECT().
		GetURL(gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewRedirect(st, ca, testLogger())

	got, err := svc.Resolve(ctx, "abc123")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if got != "https://example.com" {
		t.Fatalf("expected url %q, got %q", "https://example.com", got)
	}
}

func TestRedirectService_Resolve_CacheMiss_StoreHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	ca.EXPECT().
		Get(gomock.Any(), "zzz").
		Return("", errors.New("cache miss"))

	st.EXPECT().
		GetURL(gomock.Any(), "zzz").
		Return("https://store.com", nil)

	// write-through в кэш
	ca.EXPECT().
		Set(gomock.Any(), "zzz", "https://store.com", cacheTTL).
		Return(nil)

	svc := NewRedirect(st, ca, testLogger())

	got, err := svc.Resolve(ctx, "zzz")
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if got != "https://store.com" {
		t.Fatalf("expected url %q, got %q", "https://store.com", got)
	}
}

func TestRedirectService_Resolve_CacheMiss_StoreError_ReturnsShortNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	st := storemocks.NewMockStore(ctrl)
	ca := cachemocks.NewMockCache(ctrl)

	ca.EXPECT().
		Get(gomock.Any(), "nope").
		Return("", errors.New("cache miss"))

	st.EXPECT().
		GetURL(gomock.Any(), "nope").
		Return("", errors.New("db down"))

	// кэш-сет не должен быть вызван
	ca.EXPECT().
		Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewRedirect(st, ca, testLogger())

	_, err := svc.Resolve(ctx, "nope")
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if err != ErrShortNotFound {
		t.Fatalf("expected ErrShortNotFound, got %v", err)
	}
}
