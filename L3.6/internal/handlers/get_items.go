package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"salestracker/internal/model"
)

// ListItemsResponse — ответ на GET /items.
type ListItemsResponse struct {
	Items   []model.Item `json:"items"`
	Limit   int          `json:"limit"`
	Offset  int          `json:"offset"`
	HasMore bool         `json:"has_more"`
}

// ItemsLister возвращает страницу записей по фильтру.
type ItemsLister interface {
	List(ctx context.Context, f model.ItemFilter) ([]model.Item, bool, error)
}

// ListItems обрабатывает GET /items.
//
// Query params: from, to, type, category, sort (occurred_at|amount|id), order (asc|desc), limit, offset.
func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.ListItems"
	log := h.log.With(slog.String("op", op))

	f, err := parseItemFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	items, hasMore, err := h.lister.List(r.Context(), f)
	if err != nil {
		log.Error("failed to list items", slog.Any("err", err))
		http.Error(w, "failed to list items", http.StatusInternalServerError)
		return
	}
	if items == nil {
		items = []model.Item{}
	}

	writeJSON(w, http.StatusOK, ListItemsResponse{
		Items:   items,
		Limit:   f.Limit,
		Offset:  f.Offset,
		HasMore: hasMore,
	})
}

// parseItemFilter извлекает фильтр из query параметров.
func parseItemFilter(r *http.Request) (model.ItemFilter, error) {
	q := r.URL.Query()

	from, err := parseTimeParam(q.Get("from"))
	if err != nil {
		return model.ItemFilter{}, err
	}
	to, err := parseTimeParam(q.Get("to"))
	if err != nil {
		return model.ItemFilter{}, err
	}

	var typ model.ItemType
	if s := strings.TrimSpace(q.Get("type")); s != "" {
		t := model.ItemType(s)
		if !t.Valid() {
			return model.ItemFilter{}, errInvalidType
		}
		typ = t
	}

	cat := strings.TrimSpace(q.Get("category"))
	if len(cat) > 64 {
		return model.ItemFilter{}, errCategoryTooLong
	}

	limit, offset, err := parseLimitOffset(r)
	if err != nil {
		return model.ItemFilter{}, err
	}

	sort := strings.ToLower(strings.TrimSpace(q.Get("sort")))
	switch sort {
	case "", "occurred_at", "amount", "id":
		// ok
	default:
		return model.ItemFilter{}, errInvalidSort
	}

	order := strings.ToLower(strings.TrimSpace(q.Get("order")))
	switch order {
	case "", "asc", "desc":
		// ok
	default:
		return model.ItemFilter{}, errInvalidOrder
	}

	return model.ItemFilter{
		From:     from,
		To:       to,
		Type:     typ,
		Category: cat,
		Limit:    limit,
		Offset:   offset,
		Sort:     sort,
		Order:    order,
	}, nil
}
