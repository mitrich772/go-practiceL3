package handlers

import (
	"errors"
	"io/fs"
	"net/http"

	"image-processor/internal/repo"

	"github.com/go-chi/chi/v5"
	"log/slog"
)

// DeleteImage — DELETE /image/{id}
func (h *Handler) DeleteImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := chi.URLParam(r, "id")
	log := h.Logger.With(
		slog.String("method", "DeleteImage"),
		slog.String("id", id),
	)

	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	// 1) Читаем из БД, чтобы узнать пути файлов
	img, err := h.DB.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Error("db get failed", slog.Any("err", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// 2) Удаляем оригинал (если уже нет файла — считаем ок)
	if err := h.Storage.DeleteOriginal(ctx, img.OriginalPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Error("delete original failed",
			slog.Any("err", err),
			slog.String("path", img.OriginalPath),
		)
		http.Error(w, "failed to delete file", http.StatusInternalServerError)
		return
	}

	// 3) Удаляем обработанный (если он есть)
	if img.ProcessedPath != nil && *img.ProcessedPath != "" {
		if err := h.Storage.DeleteProcessed(ctx, *img.ProcessedPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Error("delete processed failed",
				slog.Any("err", err),
				slog.String("path", *img.ProcessedPath),
			)
			http.Error(w, "failed to delete file", http.StatusInternalServerError)
			return
		}
	}

	// 4) Удаляем запись из БД
	if err := h.DB.Delete(ctx, id); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			// редкий рейс: могли удалить параллельно
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		log.Error("db delete failed", slog.Any("err", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Info("image deleted",
		slog.String("status", string(img.Status)),
	)

	// 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
