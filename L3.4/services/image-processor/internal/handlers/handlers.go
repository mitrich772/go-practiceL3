package handlers

import (
	"log/slog"

	"github.com/wb-go/wbf/kafka"
	"github.com/wb-go/wbf/retry"

	"contracts/storage"

	"image-processor/internal/repo"
)

type Handler struct {
	Logger   *slog.Logger
	Producer *kafka.Producer
	DB       repo.ImageRepo
	Storage  storage.APIStorage

	retryStrategy  retry.Strategy
	MaxUploadBytes int64
}

func New(
	logger *slog.Logger,
	producer *kafka.Producer,
	st repo.ImageRepo,
	storage storage.APIStorage,
	kafkaRetryStrategy retry.Strategy,
	maxUploadBytes int64,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	// базовая приписка для всех логов хендлеров
	baseLog := logger.With(
		slog.String("layer", "http"),
		slog.String("component", "handlers"),
	)

	if maxUploadBytes <= 0 {
		maxUploadBytes = 10 << 20 // 10MB дефолт
	}

	return &Handler{
		Logger:         baseLog,
		Producer:       producer,
		DB:             st,
		Storage:        storage,
		retryStrategy:  kafkaRetryStrategy,
		MaxUploadBytes: maxUploadBytes,
	}
}
