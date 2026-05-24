package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"warehousecontrol/internal/model"
	"warehousecontrol/internal/repo"
)

// UpdateItemRequest описывает payload для PUT /items/{id}.
type UpdateItemRequest = CreateItemRequest

// UpdateItemResponse — ответ на успешное обновление.
type UpdateItemResponse struct {
	Item model.Item `json:"item"`
}

// ItemUpdater обновляет товары в хранилище.
type ItemUpdater interface {
	Update(ctx context.Context, item *model.Item, actor model.AuthUser) (model.Item, error)
}

// UpdateItem обрабатывает PUT /items/{id}.
func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.UpdateItem"
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

	var req UpdateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	item, err := buildItemFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	item.ID = id

	updated, err := h.updater.Update(r.Context(), &item, user)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFound):
			log.Warn("item not found", slog.Int64("id", id))
			http.Error(w, "item not found", http.StatusNotFound)
		default:
			log.Error("failed to update item", slog.Any("err", err))
			http.Error(w, "failed to update item", http.StatusInternalServerError)
		}
		return
	}

	log.Info("item updated", slog.Int64("id", updated.ID), slog.String("actor", user.Username))
	writeJSON(w, http.StatusOK, UpdateItemResponse{Item: updated})
}
