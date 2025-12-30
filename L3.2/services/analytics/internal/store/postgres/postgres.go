package postgres

import (
	"analytics/internal/handlers/dto"
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

func (s *StorePG) InsertClick(ctx context.Context, short, ip, ua, referer string) error {
	const q = `
		INSERT INTO clicks (short, ip, user_agent, referer)
		VALUES ($1, $2, $3, $4)
	`
	_, err := s.db.ExecContext(ctx, q, short, ip, ua, referer)
	return err
}

func (s *StorePG) GetStats(ctx context.Context, short string) (int64, []dto.ClickEntry, error) {

	var total int64
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM clicks WHERE short = $1`,
		short,
	).Scan(&total); err != nil {
		return 0, nil, err
	}

	rows, err := s.db.QueryContext(
		ctx,
		`SELECT clicked_at, user_agent
		 FROM clicks
		 WHERE short = $1
		 ORDER BY clicked_at DESC
		 LIMIT 100`,
		short,
	)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	var clicks []dto.ClickEntry
	for rows.Next() {
		var c dto.ClickEntry
		if err := rows.Scan(&c.ClickedAt, &c.UserAgent); err != nil {
			return 0, nil, err
		}
		clicks = append(clicks, c)
	}

	return total, clicks, nil
}
