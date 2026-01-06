// Package postgres содержит реализацию хранилища на PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"errors"

	// Register PostgreSQL driver.
	_ "github.com/lib/pq"
)

// ErrNotFound возвращается, когда запись не найдена.
var ErrNotFound = errors.New("not found")

// StorePG реализует store.Store через database/sql и PostgreSQL.
type StorePG struct {
	db *sql.DB
}

// New создаёт StorePG.
func New(db *sql.DB) *StorePG {
	return &StorePG{db: db}
}

// SaveURL сохраняет пару (short -> original) в базе данных.
func (s *StorePG) SaveURL(ctx context.Context, key string, value string) error {
	const query = `
		INSERT INTO links (short, original)
		VALUES ($1, $2)
	`

	_, err := s.db.ExecContext(ctx, query, key, value)
	return err
}

// GetURL возвращает оригинальный URL по short.
func (s *StorePG) GetURL(ctx context.Context, key string) (string, error) {
	const query = `
		SELECT original
		FROM links
		WHERE short = $1
	`

	var original string
	err := s.db.QueryRowContext(ctx, query, key).Scan(&original)
	if err == sql.ErrNoRows {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}

	return original, nil
}
