package handlers

import (
	"bufio"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"contracts/model"

	"image-processor/internal/repo"
)

// GetImage — GET /image/{id}
func (h *Handler) GetImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	// 1) БД: достаём метаданные
	img, err := h.DB.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		h.Logger.Error("db get failed", "err", err, "id", id)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 2) Polling по статусу
	switch img.Status {
	case model.StatusProcessing:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // 202
		_, _ = w.Write([]byte(`{"id":"` + id + `","status":"processing"}`))
		return

	case model.StatusFailed:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"id":"` + id + `","status":"failed"}`))
		return

	case model.StatusReady:
		// идём дальше и отдаём файл

	default:
		h.Logger.Error("unknown status", "id", id, "status", string(img.Status))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 3) Проверяем processed_path
	if img.ProcessedPath == nil || *img.ProcessedPath == "" {
		h.Logger.Error("ready but processed_path is empty", "id", id)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	processedPath := *img.ProcessedPath

	// 4) Открываем файл через storage
	rc, err := h.Storage.OpenProcessed(ctx, processedPath)
	if err != nil {
		// если storage возвращает os.ErrNotExist — можно мапить в 404
		h.Logger.Error("open processed failed", "err", err, "id", id, "path", processedPath)
		http.Error(w, "processed file not found", http.StatusNotFound)
		return
	}
	defer func() { _ = rc.Close() }()

	// 5) Content-Type: сначала по расширению
	if ext := filepath.Ext(processedPath); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
	}

	// Если по расширению не удалось — определим по первым байтам, но без потери данных
	if w.Header().Get("Content-Type") == "" {
		br := bufio.NewReader(rc)
		peek, _ := br.Peek(512)
		w.Header().Set("Content-Type", http.DetectContentType(peek))

		rc = io.NopCloser(br)
	}

	w.WriteHeader(http.StatusOK)

	// 6) Стримим в ответ
	if _, err := io.Copy(w, rc); err != nil {
		// клиент мог закрыть соединение — это не всегда “ошибка сервера”
		h.Logger.Error("stream processed failed", "err", err, "id", id)
		return
	}
}
