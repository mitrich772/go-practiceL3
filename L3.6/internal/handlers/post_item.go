package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"salestracker/internal/model"
)

// CreateItemRequest описывает payload для POST /items.
type CreateItemRequest struct {
	Type       model.ItemType `json:"type"`
	Amount     float64        `json:"amount"`
	Category   string         `json:"category"`
	Note       string         `json:"note"`
	OccurredAt string         `json:"occurred_at"` // RFC3339 или YYYY-MM-DD; пусто = now()
}

// CreateItemResponse — ответ на успешное создание.
type CreateItemResponse struct {
	Item model.Item `json:"item"`
}

// ItemCreator создаёт записи в хранилище.
type ItemCreator interface {
	Create(ctx context.Context, item *model.Item) (model.Item, error)
}

// CreateItem обрабатывает POST /items.
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.CreateItem"
	log := h.log.With(slog.String("op", op))

	var req CreateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	item, err := buildItemFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	created, err := h.creator.Create(r.Context(), &item)
	if err != nil {
		log.Error("failed to create item", slog.Any("err", err))
		http.Error(w, "failed to create item", http.StatusInternalServerError)
		return
	}

	log.Info("item created", slog.Int64("id", created.ID))
	writeJSON(w, http.StatusCreated, CreateItemResponse{Item: created})
}

// buildItemFromRequest валидирует payload и формирует доменный model.Item.
func buildItemFromRequest(req CreateItemRequest) (model.Item, error) {
	req.Category = strings.TrimSpace(req.Category)
	req.Note = strings.TrimSpace(req.Note)

	if !req.Type.Valid() {
		return model.Item{}, errInvalidType
	}
	if req.Amount < 0 {
		return model.Item{}, errInvalidAmount
	}
	if req.Category == "" {
		return model.Item{}, errCategoryRequired
	}
	if len(req.Category) > 64 {
		return model.Item{}, errCategoryTooLong
	}

	occurred := time.Now().UTC()
	if strings.TrimSpace(req.OccurredAt) != "" {
		t, err := parseTimeParam(req.OccurredAt)
		if err != nil {
			return model.Item{}, err
		}
		occurred = *t
	}

	return model.Item{
		Type:       req.Type,
		Amount:     req.Amount,
		Category:   req.Category,
		Note:       req.Note,
		OccurredAt: occurred,
	}, nil
}
