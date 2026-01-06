// Package service содержит бизнес-логику приложения.
package service

import (
	"context"

	"analytics/internal/handlers/dto"
	"analytics/internal/store"
)

// AnalyticsService реализует бизнес-логику аналитики кликов.
type AnalyticsService struct {
	store store.Store
}

// New создаёт новый AnalyticsService.
func New(store store.Store) *AnalyticsService {
	return &AnalyticsService{store: store}
}

// SaveClick сохраняет событие клика по short.
func (s *AnalyticsService) SaveClick(ctx context.Context, short, ip, ua, referer string) error {
	return s.store.InsertClick(ctx, short, ip, ua, referer)
}

// GetStats возвращает общее число кликов и последние события кликов для short.
func (s *AnalyticsService) GetStats(ctx context.Context, short string) (int64, []dto.ClickEntry, error) {
	return s.store.GetStats(ctx, short)
}
