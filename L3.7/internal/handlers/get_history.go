package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"warehousecontrol/internal/model"
)

// HistoryGetter возвращает историю изменений товара.
type HistoryGetter interface {
	History(ctx context.Context, itemID int64) ([]model.HistoryEntry, error)
}

// HistoryResponse — ответ на GET /items/{id}/history.
type HistoryResponse struct {
	History []model.HistoryEntry `json:"history"`
}

// GetHistory обрабатывает GET /items/{id}/history.
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.GetHistory"
	log := h.log.With(slog.String("op", op))

	id, err := parseIDPath(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	history, err := h.history.History(r.Context(), id)
	if err != nil {
		log.Error("failed to get history", slog.Any("err", err))
		http.Error(w, "failed to get history", http.StatusInternalServerError)
		return
	}
	if history == nil {
		history = []model.HistoryEntry{}
	}
	writeJSON(w, http.StatusOK, HistoryResponse{History: history})
}
