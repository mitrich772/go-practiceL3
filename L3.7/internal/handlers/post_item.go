package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"warehousecontrol/internal/model"
)

// CreateItemRequest описывает payload для POST /items.
type CreateItemRequest struct {
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	Quantity    int    `json:"quantity"`
	Location    string `json:"location"`
	Description string `json:"description"`
}

// CreateItemResponse — ответ на успешное создание.
type CreateItemResponse struct {
	Item model.Item `json:"item"`
}

// ItemCreator создаёт товары в хранилище.
type ItemCreator interface {
	Create(ctx context.Context, item *model.Item, actor model.AuthUser) (model.Item, error)
}

// CreateItem обрабатывает POST /items.
func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.CreateItem"
	log := h.log.With(slog.String("op", op))
	user, ok := requireUser(w, r)
	if !ok {
		return
	}

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

	created, err := h.creator.Create(r.Context(), &item, user)
	if err != nil {
		log.Error("failed to create item", slog.Any("err", err))
		http.Error(w, "failed to create item", http.StatusInternalServerError)
		return
	}

	log.Info("item created", slog.Int64("id", created.ID), slog.String("actor", user.Username))
	writeJSON(w, http.StatusCreated, CreateItemResponse{Item: created})
}

// buildItemFromRequest валидирует payload и формирует доменный model.Item.
func buildItemFromRequest(req CreateItemRequest) (model.Item, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.SKU = strings.TrimSpace(req.SKU)
	req.Location = strings.TrimSpace(req.Location)
	req.Description = strings.TrimSpace(req.Description)

	if req.Name == "" || len(req.Name) > 120 {
		return model.Item{}, errInvalidName
	}
	if req.SKU == "" || len(req.SKU) > 64 {
		return model.Item{}, errInvalidSKU
	}
	if req.Quantity < 0 {
		return model.Item{}, errInvalidQuantity
	}
	if len(req.Location) > 120 {
		return model.Item{}, errInvalidLocation
	}

	return model.Item{
		Name:        req.Name,
		SKU:         req.SKU,
		Quantity:    req.Quantity,
		Location:    req.Location,
		Description: req.Description,
	}, nil
}
