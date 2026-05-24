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

// ItemGetter возвращает товар по ID.
type ItemGetter interface {
	GetByID(ctx context.Context, id int64) (model.Item, error)
}

// GetItemResponse — ответ на GET /items/{id}.
type GetItemResponse struct {
	Item model.Item `json:"item"`
}

// GetItem обрабатывает GET /items/{id}.
func (h *Handler) GetItem(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.GetItem"
	log := h.log.With(slog.String("op", op))

	id, err := parseIDPath(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	item, err := h.getter.GetByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFound):
			http.Error(w, "item not found", http.StatusNotFound)
		default:
			log.Error("failed to get item", slog.Any("err", err))
			http.Error(w, "failed to get item", http.StatusInternalServerError)
		}
		return
	}

	writeJSON(w, http.StatusOK, GetItemResponse{Item: item})
}
