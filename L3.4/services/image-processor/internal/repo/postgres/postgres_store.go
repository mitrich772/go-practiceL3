package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/wb-go/wbf/dbpg"

	"contracts/model"

	"image-processor/internal/repo"
)

type PostgresRepo struct {
	db  *dbpg.DB
	log *slog.Logger
}

func New(db *dbpg.DB, log *slog.Logger) *PostgresRepo {
	if log == nil {
		log = slog.Default()
	}

	// базовая приписка для всех логов БД
	baseLog := log.With(
		slog.String("layer", "db"),
		slog.String("repo", "postgres"),
	)

	return &PostgresRepo{
		db:  db,
		log: baseLog,
	}
}

// Create — вставляет запись в таблицу images.
func (r *PostgresRepo) Create(ctx context.Context, img model.Image) error {
	log := r.log.With(slog.String("method", "Create"))
	start := time.Now()

	// дефолты
	now := time.Now()
	if img.CreatedAt.IsZero() {
		img.CreatedAt = now
	}
	if img.UpdatedAt.IsZero() {
		img.UpdatedAt = img.CreatedAt
	}
	if img.Status == "" {
		img.Status = model.StatusProcessing
	}

	const q = `
		INSERT INTO images (id, status, original_path, processed_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.ExecContext(
		ctx,
		q,
		img.ID,
		string(img.Status),
		img.OriginalPath,
		img.ProcessedPath,
		img.CreatedAt,
		img.UpdatedAt,
	)
	if err != nil {
		log.Error("db create failed",
			slog.Any("err", err),
			slog.String("id", img.ID),
			slog.String("status", string(img.Status)),
			slog.String("original_path", img.OriginalPath),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	log.Info("db create ok",
		slog.String("id", img.ID),
		slog.String("status", string(img.Status)),
		slog.Duration("took", time.Since(start)),
	)
	return nil
}

// Delete — удаляет запись по id. Если строк нет — ErrNotFound.
func (r *PostgresRepo) Delete(ctx context.Context, id string) error {
	log := r.log.With(slog.String("method", "Delete"))
	start := time.Now()

	const q = `DELETE FROM images WHERE id = $1`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		log.Error("db delete failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		log.Error("db delete rows_affected failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	if affected == 0 {
		log.Info("db delete not found",
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return repo.ErrNotFound
	}

	log.Info("db delete ok",
		slog.String("id", id),
		slog.Int64("rows", affected),
		slog.Duration("took", time.Since(start)),
	)
	return nil
}

func (r *PostgresRepo) Get(ctx context.Context, id string) (model.Image, error) {
	log := r.log.With(slog.String("method", "Get"))
	start := time.Now()

	const q = `
		SELECT id, status, original_path, processed_path, created_at, updated_at
		FROM images
		WHERE id = $1
	`

	var (
		img       model.Image
		statusStr string
		procNS    sql.NullString
	)

	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&img.ID,
		&statusStr,
		&img.OriginalPath,
		&procNS,
		&img.CreatedAt,
		&img.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("db get not found",
				slog.String("id", id),
				slog.Duration("took", time.Since(start)),
			)
			return model.Image{}, repo.ErrNotFound
		}

		log.Error("db get failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return model.Image{}, err
	}

	img.Status = model.Status(statusStr)

	if procNS.Valid {
		v := procNS.String
		img.ProcessedPath = &v
	} else {
		img.ProcessedPath = nil
	}

	log.Info("db get ok",
		slog.String("id", img.ID),
		slog.String("status", string(img.Status)),
		slog.Duration("took", time.Since(start)),
	)
	return img, nil
}

