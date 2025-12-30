package postgres

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
)

var ErrNotFound = errors.New("not found")

type StorePG struct {
	db *sql.DB
}

func New(db *sql.DB) *StorePG {
	return &StorePG{db: db}
}

func (s *StorePG) SaveURL(ctx context.Context, key string, value string) error {
	const query = `
		INSERT INTO links (short, original)
		VALUES ($1, $2)
	`

	_, err := s.db.ExecContext(ctx, query, key, value)
	return err
}

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
