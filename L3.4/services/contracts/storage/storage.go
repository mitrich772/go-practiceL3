package storage

import (
	"context"
	"io"
)

// APIStorage — зависимости для HTTP API.
type APIStorage interface {
	SaveOriginal(ctx context.Context, id string, filename string, data []byte) (string, error)
	OpenProcessed(ctx context.Context, path string) (io.ReadCloser, error)
	SaveProcessed(ctx context.Context, id string, ext string, data []byte) (string, error) // опционально: если API когда-то будет писать processed
	DeleteOriginal(ctx context.Context, path string) error
	DeleteProcessed(ctx context.Context, path string) error
}

// WorkerStorage — зависимости для worker.
type WorkerStorage interface {
	OpenOriginal(ctx context.Context, path string) (io.ReadCloser, error)
	SaveProcessed(ctx context.Context, id string, ext string, data []byte) (string, error)
	DeleteOriginal(ctx context.Context, path string) error
}
