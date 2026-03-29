package service

import (
	"context"
	"errors"

	"contracts/dto"
)

var (
	ErrUnsupportedMode = errors.New("unsupported image processing mode")
)

// ImageProcessor описывает контракт сервиса обработки изображений.
type ImageProcessor interface {
	// Process принимает сырые байты оригинальной картинки и DTO-задачу,
	// возвращает новые (обработанные) байты, новое расширение файла (".jpg", ".png" и т.д.)
	// или ошибку, если обработка не удалась.
	Process(ctx context.Context, origBytes []byte, job dto.ImageJob) ([]byte, string, error)
}
