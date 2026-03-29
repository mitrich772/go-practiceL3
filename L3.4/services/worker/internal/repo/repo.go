package repo

import (
	"context"
	"errors"

	"contracts/model"
)

var (
	// ErrNotFound — запись не найдена.
	ErrNotFound = errors.New("not found")
)

type WorkerImageRepo interface {
	Get(ctx context.Context, id string) (model.Image, error) // optional
	MarkReady(ctx context.Context, id string, processedPath string) error
	MarkFailed(ctx context.Context, id string) error
}
