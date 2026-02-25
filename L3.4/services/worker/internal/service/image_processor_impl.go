package service

import (
	"bytes"
	"context"
	"contracts/dto"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/fogleman/gg"
)

type imageProcessorService struct {
	log *slog.Logger
}

func NewImageProcessor(log *slog.Logger) ImageProcessor {
	if log == nil {
		log = slog.Default()
	}

	// Можно добавить какие-то дефолтные конфиги,
	// лимиты памяти или внешние зависимости (например, кэш).
	return &imageProcessorService{
		log: log.With(slog.String("component", "ImageProcessor")),
	}
}

// Process — главная точка входа. Она определяет, какой режим обработки выбран,
// и делегирует работу соответствующему методу.
func (s *imageProcessorService) Process(ctx context.Context, origBytes []byte, job dto.ImageJob) ([]byte, string, error) {
	// 1. Извлекаем расширение оригинального файла (чтобы вернуть его же, если формат не меняется)
	ext := filepath.Ext(job.OriginalPath)
	if ext == "" {
		ext = ".jpg"
	}
	ext = strings.ToLower(ext)
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	var processedBytes []byte
	var err error

	// 2. Делегируем логику конкретному методу в зависимости от режима
	switch job.Mode {
	case "resize":
		processedBytes, err = s.resize(ctx, origBytes, job.Width, job.Height, ext)
	case "thumb":
		processedBytes, err = s.thumb(ctx, origBytes, job.Width, job.Height, ext)
	case "watermark":
		processedBytes, err = s.watermark(ctx, origBytes, job.WatermarkText, ext)
	default:
		s.log.Error("unsupported mode", slog.String("mode", job.Mode), slog.String("id", job.ID))
		return nil, "", ErrUnsupportedMode
	}

	if err != nil {
		return nil, "", err
	}

	return processedBytes, ext, nil
}

// decodeImage читает исходные байты и превращает их в image.Image
func (s *imageProcessorService) decodeImage(imgBytes []byte) (image.Image, string, error) {
	img, format, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}
	return img, format, nil
}

// encodeImage сохраняет картинку обратно в байты в нужном формате (.jpg или .png)
func (s *imageProcessorService) encodeImage(img image.Image, ext string) ([]byte, error) {
	buf := new(bytes.Buffer)
	var err error

	switch ext {
	case ".png":
		err = imaging.Encode(buf, img, imaging.PNG)
	default:
		err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(90))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	return buf.Bytes(), nil
}

func (s *imageProcessorService) resize(ctx context.Context, imgBytes []byte, width, height int, ext string) ([]byte, error) {
	s.log.Info("processing resize", slog.Int("w", width), slog.Int("h", height))

	img, _, err := s.decodeImage(imgBytes)
	if err != nil {
		return nil, err
	}

	if width <= 0 && height <= 0 {
		return imgBytes, nil
	}

	resImg := imaging.Resize(img, width, height, imaging.Lanczos)

	return s.encodeImage(resImg, ext)
}

func (s *imageProcessorService) thumb(ctx context.Context, imgBytes []byte, width, height int, ext string) ([]byte, error) {
	s.log.Info("processing thumb", slog.Int("w", width), slog.Int("h", height))

	img, _, err := s.decodeImage(imgBytes)
	if err != nil {
		return nil, err
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("width and height must be > 0 for thumb mode")
	}

	thumbImg := imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)

	return s.encodeImage(thumbImg, ext)
}

func (s *imageProcessorService) watermark(ctx context.Context, imgBytes []byte, watermarkText string, ext string) ([]byte, error) {
	s.log.Info("processing watermark", slog.String("text", watermarkText))

	img, _, err := s.decodeImage(imgBytes)
	if err != nil {
		return nil, err
	}

	if watermarkText == "" {
		watermarkText = "WATERMARK"
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	dc := gg.NewContext(w, h)
	dc.DrawImage(img, 0, 0)

	dc.SetColor(color.RGBA{255, 255, 255, 128})

	dc.DrawLine(0, 0, float64(w), float64(h))
	dc.SetLineWidth(10)
	dc.SetColor(color.RGBA{255, 0, 0, 100})
	dc.Stroke()

	dc.SetColor(color.RGBA{255, 255, 255, 200})
	dc.DrawStringAnchored(watermarkText, float64(w)/2, float64(h)/2, 0.5, 0.5)

	outImg := dc.Image()

	return s.encodeImage(outImg, ext)
}
