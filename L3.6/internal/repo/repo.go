// Package repo описывает контракты доступа к хранилищу SalesTracker.
package repo

import (
	"context"
	"errors"

	"salestracker/internal/model"
)

// ErrNotFound возвращается, когда запись с указанным ID не существует.
var ErrNotFound = errors.New("item not found")

// Repo — общий интерфейс репозитория, объединяющий все операции.
// Хендлеры зависят от отдельных verb-er интерфейсов (см. internal/handlers).
type Repo interface {
	Create(ctx context.Context, item *model.Item) (model.Item, error)
	Update(ctx context.Context, item *model.Item) (model.Item, error)
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (model.Item, error)
	List(ctx context.Context, f model.ItemFilter) ([]model.Item, bool, error)
	Analytics(ctx context.Context, f model.ItemFilter) (model.Analytics, error)
}
