// Package store описывает интерфейсы слоя хранения данных.
package store

import "context"

//go:generate mockgen -source=store.go -destination=./mocks/mock_store.go -package=storemocks

// Store описывает операции хранилища ссылок.
type Store interface {
	GetURL(ctx context.Context, short string) (string, error)
	SaveURL(ctx context.Context, short string, original string) error
}
