package store

import (
	"context"
)

type Store interface {
	GetURL(ctx context.Context, short string) (string, error)
	SaveURL(ctx context.Context, short string, original string) error
}
