package postgres

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/wb-go/wbf/dbpg"

	"contracts/model"

	"worker/internal/repo"
)

type PostgresRepo struct {
	db  *dbpg.DB
	log *slog.Logger
}

func New(db *dbpg.DB, log *slog.Logger) *PostgresRepo {
	if log == nil {
		log = slog.Default()
	}

	baseLog := log.With(
		slog.String("layer", "db"),
		slog.String("repo", "postgres"),
	)

	return &PostgresRepo{
		db:  db,
		log: baseLog,
	}
}

var _ repo.WorkerImageRepo = (*PostgresRepo)(nil)

// Get — получить запись по id.
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

// MarkReady — отметить обработку успешной и записать processed_path.
// Делаем update только если status=processing, чтобы не перетереть failed/ready.
func (r *PostgresRepo) MarkReady(ctx context.Context, id string, processedPath string) error {
	log := r.log.With(slog.String("method", "MarkReady"))
	start := time.Now()

	const q = `
		UPDATE images
		SET status = 'ready',
		    processed_path = $2,
		    updated_at = NOW()
		WHERE id = $1 AND status = 'processing'
	`

	res, err := r.db.ExecContext(ctx, q, id, processedPath)
	if err != nil {
		log.Error("db mark_ready update failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.String("processed_path", processedPath),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		log.Error("db mark_ready rows_affected failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	if affected == 1 {
		log.Info("db mark_ready ok",
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return nil
	}

	// если не обновили — выясняем почему: нет записи или уже другой статус
	const q2 = `SELECT status FROM images WHERE id = $1`

	var statusStr string
	err = r.db.QueryRowContext(ctx, q2, id).Scan(&statusStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("db mark_ready not found",
				slog.String("id", id),
				slog.Duration("took", time.Since(start)),
			)
			return repo.ErrNotFound
		}
		log.Error("db mark_ready select failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	// уже ready/failed — ок, просто пропускаем
	log.Info("db mark_ready skipped (status not processing)",
		slog.String("id", id),
		slog.String("status", statusStr),
		slog.Duration("took", time.Since(start)),
	)

	return nil
}

// MarkFailed — отметить обработку как failed (только если status=processing).
func (r *PostgresRepo) MarkFailed(ctx context.Context, id string) error {
	log := r.log.With(slog.String("method", "MarkFailed"))
	start := time.Now()

	const q = `
		UPDATE images
		SET status = 'failed', updated_at = NOW()
		WHERE id = $1 AND status = 'processing'
	`

	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		log.Error("db mark_failed update failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		log.Error("db mark_failed rows_affected failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	if affected == 1 {
		log.Info("db mark_failed ok",
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return nil
	}

	const q2 = `SELECT status FROM images WHERE id = $1`

	var statusStr string
	err = r.db.QueryRowContext(ctx, q2, id).Scan(&statusStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("db mark_failed not found",
				slog.String("id", id),
				slog.Duration("took", time.Since(start)),
			)
			return repo.ErrNotFound
		}
		log.Error("db mark_failed select failed",
			slog.Any("err", err),
			slog.String("id", id),
			slog.Duration("took", time.Since(start)),
		)
		return err
	}

	log.Info("db mark_failed skipped (status not processing)",
		slog.String("id", id),
		slog.String("status", statusStr),
		slog.Duration("took", time.Since(start)),
	)

	return nil
}
