package kafka_consumer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"contracts/dto"
	"contracts/model"
	contractStorage "contracts/storage"
	"worker/internal/repo"

	"github.com/segmentio/kafka-go"
	wbfkafka "github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"
)

type WorkerConsumer struct {
	consumer   *wbfkafka.Consumer
	log        *slog.Logger
	fetchRetry retry.Strategy

	storage contractStorage.WorkerStorage
	repo    repo.WorkerImageRepo

	// commitRetry retry.Strategy // TODO: опционально, если захочешь retry на commit
}

// New — конструктор consumer'а воркера.
func New(
	log *slog.Logger,
	consumer *wbfkafka.Consumer,
	fetchRetry retry.Strategy,
	st contractStorage.WorkerStorage,
	rp repo.WorkerImageRepo,
) *WorkerConsumer {
	if log == nil {
		log = slog.Default()
	}

	// дефолтный retry на fetch
	if fetchRetry.Attempts <= 0 {
		fetchRetry = retry.Strategy{
			Attempts: 10,
			Delay:    1 * time.Second,
			Backoff:  2,
		}
	}

	base := log.With(
		slog.String("layer", "transport"),
		slog.String("component", "kafka.consumer"),
		slog.String("class", "WorkerConsumer"),
	)

	return &WorkerConsumer{
		consumer:   consumer,
		log:        base,
		fetchRetry: fetchRetry,
		storage:    st,
		repo:       rp,
	}
}

// TODO: вынести в сервис и логи левел debug бахнуть и будет супир
// StartConsume — минимально рабочий цикл.
// Сейчас: read original -> save processed (как есть) -> MarkReady -> commit.
func (c *WorkerConsumer) StartConsume(ctx context.Context) {
	l := c.log.With(slog.String("method", "StartConsume"))

	if c.consumer == nil {
		l.Error("consumer is nil")
		return
	}
	if c.storage == nil {
		l.Error("storage is nil")
		return
	}
	if c.repo == nil {
		l.Error("repo is nil")
		return
	}

	l.Info("consumer started")

	for {
		select {
		case <-ctx.Done():
			l.Info("consumer stopped", slog.Any("err", ctx.Err()))
			return
		default:
		}

		msg, err := c.consumer.FetchWithRetry(ctx, c.fetchRetry)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			l.Error("fetch failed", slog.Any("err", err))
			continue
		}

		job, ok := c.parseJobOrCommitBad(ctx, l, msg)
		if !ok {
			continue
		}

		// 1) проверяем запись в БД (если удалили — skip + commit)
		img, err := c.repo.Get(ctx, job.ID)
		if err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				_ = c.commitAndLog(ctx, l, msg, "skip: deleted")
				continue
			}
			l.Error("db get failed", slog.Any("err", err), slog.String("id", job.ID))
			continue // НЕ commit -> Kafka даст повтор
		}

		// 2) если уже не processing — смысла обрабатывать нет
		switch img.Status {
		case model.StatusProcessing:
			// ok
		case model.StatusReady:
			_ = c.commitAndLog(ctx, l, msg, "skip: already ready")
			continue
		case model.StatusFailed:
			_ = c.commitAndLog(ctx, l, msg, "skip: already failed")
			continue
		default:
			// непонятный статус — лучше НЕ коммитить, чтобы не потерять задачу
			l.Error("unknown status", slog.String("id", job.ID), slog.String("status", string(img.Status)))
			continue
		}

		// 3) открыть оригинал
		rc, err := c.storage.OpenOriginal(ctx, job.OriginalPath)
		if err != nil {
			l.Error("open original failed",
				slog.Any("err", err),
				slog.String("id", job.ID),
				slog.String("path", job.OriginalPath),
			)
			c.markFailedBestEffort(ctx, l, job.ID, "open original failed")
			_ = c.commitAndLog(ctx, l, msg, "failed: open original")
			continue
		}

		origBytes, readErr := io.ReadAll(rc)
		_ = rc.Close()
		if readErr != nil {
			l.Error("read original failed", slog.Any("err", readErr), slog.String("id", job.ID))
			c.markFailedBestEffort(ctx, l, job.ID, "read original failed")
			_ = c.commitAndLog(ctx, l, msg, "failed: read original")
			continue
		}

		// 4) обработка (ПОКА STUB: просто копия)
		outBytes, outExt, procErr := c.processStub(job, origBytes)
		if procErr != nil {
			l.Error("process failed", slog.Any("err", procErr), slog.String("id", job.ID))
			c.markFailedBestEffort(ctx, l, job.ID, "process failed")
			_ = c.commitAndLog(ctx, l, msg, "failed: process")
			continue
		}

		// 5) сохранить processed
		processedPath, err := c.storage.SaveProcessed(ctx, job.ID, outExt, outBytes)
		if err != nil {
			l.Error("save processed failed", slog.Any("err", err), slog.String("id", job.ID))
			c.markFailedBestEffort(ctx, l, job.ID, "save processed failed")
			_ = c.commitAndLog(ctx, l, msg, "failed: save processed")
			continue
		}

		// 6) MarkReady
		if err := c.repo.MarkReady(ctx, job.ID, processedPath); err != nil {
			if errors.Is(err, repo.ErrNotFound) {
				// запись удалили после обработки — job больше не нужен
				_ = c.commitAndLog(ctx, l, msg, "skip: deleted after processing")
				continue
			}
			// БД временно легла -> НЕ commit (Kafka даст повтор),
			// но учти: повторная обработка должна быть идемпотентной
			l.Error("mark ready failed", slog.Any("err", err), slog.String("id", job.ID))
			continue
		}

		// 7) commit успеха
		_ = c.commitAndLog(ctx, l, msg, "processed")
	}
}

func (c *WorkerConsumer) Close() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	l := c.log.With(slog.String("method", "Close"))
	l.Info("consumer closing")
	return c.consumer.Close()
}

// parseJobOrCommitBad: парсит ImageJob. Битое/невалидное — коммитим, чтобы не зациклиться.
func (c *WorkerConsumer) parseJobOrCommitBad(
	ctx context.Context,
	l *slog.Logger,
	msg kafka.Message,
) (job dto.ImageJob, ok bool) {
	if err := json.Unmarshal(msg.Value, &job); err != nil {
		l.Error("bad message: unmarshal failed", slog.Any("err", err))
		_ = c.commitAndLog(ctx, l, msg, "bad message: unmarshal failed")
		return dto.ImageJob{}, false
	}
	if job.ID == "" || job.OriginalPath == "" || job.Mode == "" {
		l.Error("bad message: missing required fields",
			slog.String("id", job.ID),
			slog.String("original_path", job.OriginalPath),
			slog.String("mode", job.Mode),
		)
		_ = c.commitAndLog(ctx, l, msg, "bad message: missing fields")
		return dto.ImageJob{}, false
	}
	return job, true
}

// commitAndLog: commit + один лог.
func (c *WorkerConsumer) commitAndLog(ctx context.Context, l *slog.Logger, msg kafka.Message, reason string) error {
	if err := c.consumer.Commit(ctx, msg); err != nil {
		l.Error("commit failed",
			slog.Any("err", err),
			slog.String("reason", reason),
			slog.String("topic", msg.Topic),
			slog.Int("partition", msg.Partition),
			slog.Int64("offset", msg.Offset),
		)
		return err
	}
	l.Info("committed",
		slog.String("reason", reason),
		slog.String("topic", msg.Topic),
		slog.Int("partition", msg.Partition),
		slog.Int64("offset", msg.Offset),
	)
	return nil
}

// markFailedBestEffort: пытаемся пометить failed, но не ломаем поток если не получилось.
func (c *WorkerConsumer) markFailedBestEffort(ctx context.Context, l *slog.Logger, id string, reason string) {
	if err := c.repo.MarkFailed(ctx, id); err != nil && !errors.Is(err, repo.ErrNotFound) {
		l.Error("mark failed failed", slog.Any("err", err), slog.String("id", id), slog.String("reason", reason))
	}
}

// processStub: временно “обработка” = копия.
// TODO: заменить на реальную обработку (decode -> resize/thumb/watermark -> encode).
func (c *WorkerConsumer) processStub(job dto.ImageJob, in []byte) ([]byte, string, error) {
	ext := filepath.Ext(job.OriginalPath)
	if ext == "" {
		ext = ".jpg"
	}
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return in, ext, nil
}
