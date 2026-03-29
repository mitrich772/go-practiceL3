package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	contractDTO "contracts/dto"
	"contracts/model"

	"image-processor/internal/dto"
)

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Локальный логгер для этого запроса
	log := h.Logger.With(slog.String("method", "Upload"))

	// 0) Ограничиваем размер тела запроса (защита от огромных upload)
	r.Body = http.MaxBytesReader(w, r.Body, h.MaxUploadBytes)

	// 1) Парсим multipart/form-data (чтобы стали доступны FormFile/FormValue)
	if err := r.ParseMultipartForm(h.MaxUploadBytes); err != nil {
		log.Info("bad multipart form", slog.Any("err", err))
		http.Error(w, "bad multipart form", http.StatusBadRequest)
		return
	}

	// 2) Достаём файл из поля "file"
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Info("file is missing", slog.Any("err", err))
		http.Error(w, "file is required (field name: file)", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()

	filename := header.Filename
	log = log.With(slog.String("filename", filename))

	// 3) Читаем файл в память (для учебного проекта норм; позже можно потоково писать на диск)
	data, err := io.ReadAll(file)
	if err != nil {
		log.Error("failed to read file", slog.Any("err", err))
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}

	sizeBytes := int64(len(data))
	log = log.With(slog.Int64("size_bytes", sizeBytes))

	// 4) Проверяем размер файла
	if sizeBytes > h.MaxUploadBytes {
		log.Info("file too large",
			slog.Int64("max_upload_bytes", h.MaxUploadBytes),
		)
		http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
		return
	}

	// 5) Определяем Content-Type (берём из header, если пусто — определяем по байтам)
	ct := header.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	log = log.With(slog.String("content_type", ct))

	// 6) Собираем DTO из файла + полей формы (mode, width/height, watermark_text)
	reqDTO := dto.UploadRequestDTO{
		Filename:    filename,
		ContentType: ct,
		SizeBytes:   sizeBytes,
		Data:        data,

		Mode:          strings.ToLower(strings.TrimSpace(r.FormValue("mode"))),
		WatermarkText: r.FormValue("watermark_text"),
	}

	// 7) Парсим width/height (если пришли) и валидируем
	if s := strings.TrimSpace(r.FormValue("width")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			log.Info("invalid width", slog.String("width", s))
			http.Error(w, "width must be positive int", http.StatusBadRequest)
			return
		}
		reqDTO.Width = v
	}
	if s := strings.TrimSpace(r.FormValue("height")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 {
			log.Info("invalid height", slog.String("height", s))
			http.Error(w, "height must be positive int", http.StatusBadRequest)
			return
		}
		reqDTO.Height = v
	}

	// 8) Валидируем mode + проставляем дефолт
	if reqDTO.Mode == "" {
		reqDTO.Mode = "resize"
	}
	switch reqDTO.Mode {
	case "resize", "thumb", "watermark":
		// ok
	default:
		log.Info("invalid mode", slog.String("mode", reqDTO.Mode))
		http.Error(w, "mode must be one of: resize, thumb, watermark", http.StatusBadRequest)
		return
	}
	log = log.With(
		slog.String("mode", reqDTO.Mode),
		slog.Int("width", reqDTO.Width),
		slog.Int("height", reqDTO.Height),
	)

	// 9) Генерируем уникальный id (128-bit random -> hex)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		log.Error("failed to generate id", slog.Any("err", err))
		http.Error(w, "failed to generate id", http.StatusInternalServerError)
		return
	}
	id := hex.EncodeToString(idBytes)
	log = log.With(slog.String("id", id))

	// 10) Сохраняем оригинал в файловое хранилище
	originalPath, err := h.Storage.SaveOriginal(ctx, id, reqDTO.Filename, reqDTO.Data)
	if err != nil {
		log.Error("failed to save original", slog.Any("err", err))
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	log = log.With(slog.String("original_path", originalPath))

	now := time.Now()

	// 11) Пишем метаданные в БД (status=processing + original_path)
	img := model.Image{
		ID:           id,
		Status:       model.StatusProcessing,
		OriginalPath: originalPath,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := h.DB.Create(ctx, img); err != nil {
		// rollback: если БД не записалась — удаляем файл
		_ = h.Storage.DeleteOriginal(ctx, originalPath)

		log.Error("failed to create db record", slog.Any("err", err))
		http.Error(w, "failed to write db", http.StatusInternalServerError)
		return
	}

	// 12) Формируем job для воркера (что обработать и как)
	job := contractDTO.ImageJob{
		ID:            id,
		OriginalPath:  originalPath,
		Mode:          reqDTO.Mode,
		Width:         reqDTO.Width,
		Height:        reqDTO.Height,
		WatermarkText: reqDTO.WatermarkText,
	}

	// 13) Сериализуем job в JSON для Kafka
	payload, err := json.Marshal(job)
	if err != nil {
		// rollback: удалить запись из БД + удалить файл
		_ = h.DB.Delete(ctx, id)
		_ = h.Storage.DeleteOriginal(ctx, originalPath)

		log.Error("failed to marshal kafka job", slog.Any("err", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 14) Отправляем job в Kafka (topic images.in)
	if err := h.Producer.SendWithRetry(ctx, h.retryStrategy, []byte(id), payload); err != nil {
		// rollback: удалить запись из БД + удалить файл
		_ = h.DB.Delete(ctx, id)
		_ = h.Storage.DeleteOriginal(ctx, originalPath)

		log.Error("failed to send kafka job", slog.Any("err", err))
		http.Error(w, "failed to enqueue job", http.StatusInternalServerError)
		return
	}

	// 15) Успех: приняли, сохранили, поставили в очередь
	log.Info("upload accepted")

	// 16) Ответ клиенту: приняли на обработку, возвращаем id
	resp := dto.UploadResponseDTO{
		ID:        id,
		Status:    "processing",
		CreatedAt: now,

		Filename:    reqDTO.Filename,
		ContentType: reqDTO.ContentType,
		SizeBytes:   reqDTO.SizeBytes,
		Mode:        reqDTO.Mode,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(resp)
}
