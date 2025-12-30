package store

import (
	"analytics/internal/handlers/dto"
	"context"
)

type Store interface {
	InsertClick(ctx context.Context, short, ip, ua, referer string) error
	GetStats(ctx context.Context, short string) (int64, []dto.ClickEntry, error)
}
