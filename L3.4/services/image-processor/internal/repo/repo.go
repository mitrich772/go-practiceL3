package repo

import (
	"context"
	"contracts/model"
	"errors"
)

var (
	// ErrNotFound — запись не найдена.
	ErrNotFound = errors.New("not found")
)

type ImageRepo interface {
	// Create — создать запись при загрузке картинки (status=processing).
	Create(ctx context.Context, img model.Image) error

	// Get — получить запись по id (для GET /image/{id}, DELETE /image/{id}).
	Get(ctx context.Context, id string) (model.Image, error)

	// Delete — удалить запись (DELETE /image/{id}).
	Delete(ctx context.Context, id string) error

	// MarkFailed — отметить задачу упавшей (если обработка не удалась).
	MarkFailed(ctx context.Context, id string) error
}
