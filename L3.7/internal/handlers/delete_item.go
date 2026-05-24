package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"warehousecontrol/internal/model"
	"warehousecontrol/internal/repo"
)

// ItemDeleter удаляет товары по ID.
type ItemDeleter interface {
	Delete(ctx context.Context, id int64, actor model.AuthUser) error
}

// DeleteItem обрабатывает DELETE /items/{id}.
func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.DeleteItem"
	log := h.log.With(slog.String("op", op))
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

	id, err := parseIDPath(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.deleter.Delete(r.Context(), id, user); err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFound):
			log.Warn("item not found for delete", slog.Int64("id", id))
			http.Error(w, "item not found", http.StatusNotFound)
		default:
			log.Error("failed to delete item", slog.Any("err", err))
			http.Error(w, "failed to delete item", http.StatusInternalServerError)
		}
		return
	}

	log.Info("item deleted", slog.Int64("id", id), slog.String("actor", user.Username))
	w.WriteHeader(http.StatusNoContent)
}
