// Package repo описывает контракты доступа к хранилищу WarehouseControl.
package repo

import (
	"context"
	"errors"

	"warehousecontrol/internal/model"
)

// ErrNotFound возвращается, когда запись с указанным ID не существует.
var ErrNotFound = errors.New("item not found")

// Repo — общий интерфейс репозитория, объединяющий все операции.
type Repo interface {
	Create(ctx context.Context, item *model.Item, actor model.AuthUser) (model.Item, error)
	Update(ctx context.Context, item *model.Item, actor model.AuthUser) (model.Item, error)
	Delete(ctx context.Context, id int64, actor model.AuthUser) error
	GetByID(ctx context.Context, id int64) (model.Item, error)
	List(ctx context.Context, f model.ItemFilter) ([]model.Item, bool, error)
	History(ctx context.Context, itemID int64) ([]model.HistoryEntry, error)
	ListUsers(ctx context.Context) ([]model.User, error)
}
