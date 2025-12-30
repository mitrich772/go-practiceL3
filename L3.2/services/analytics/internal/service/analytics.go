package service

import (
	"context"

	"analytics/internal/handlers/dto"
	"analytics/internal/store"
)

type AnalyticsService struct {
	store store.Store
}

func New(store store.Store) *AnalyticsService {
	return &AnalyticsService{store: store}
}

func (s *AnalyticsService) SaveClick(ctx context.Context, short, ip, ua, referer string) error {
	return s.store.InsertClick(ctx, short, ip, ua, referer)
}

func (s *AnalyticsService) GetStats(ctx context.Context, short string) (int64, []dto.ClickEntry, error) {
	return s.store.GetStats(ctx, short)
}
