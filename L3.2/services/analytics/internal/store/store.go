// Package store описывает интерфейсы слоя хранения данных.
package store

import (
	"context"

	"analytics/internal/handlers/dto"
)

// Store описывает операции, необходимые сервисам для работы с хранилищем.
type Store interface {
	InsertClick(ctx context.Context, short, ip, ua, referer string) error
	GetStats(ctx context.Context, short string) (int64, []dto.ClickEntry, error)
}
