package handlers

import (
	"commenttree/internal/dto"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

type GetRootCommentsResponse struct {
	Items   []dto.CommentNode `json:"items"`
	Limit   int               `json:"limit"`
	Offset  int               `json:"offset"`
	HasMore bool              `json:"has_more"`
}
type RootCommentsGetter interface {
	ListRoots(ctx context.Context, q dto.GetRootCommentsQuery) ([]dto.CommentNode, bool, error)
}

// GET /roots?limit=20&offset=0&sort=created_at&order=desc
func (h *Handler) GetRootCommments(w http.ResponseWriter, r *http.Request) {
	// defaults
	q := dto.GetRootCommentsQuery{
		Limit:  20,
		Offset: 0,
		Sort:   "created_at",
		Order:  "desc",
	}

	if s := r.URL.Query().Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 || v > 100 {
			http.Error(w, "invalid limit (1..100)", http.StatusBadRequest)
			return
		}
		q.Limit = v
	}

	// offset
	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			http.Error(w, "invalid offset (>=0)", http.StatusBadRequest)
			return
		}
		q.Offset = v
	}

	// sort whitelist
	if s := r.URL.Query().Get("sort"); s != "" {
		s = strings.ToLower(s)
		if s != "created_at" && s != "id" {
			http.Error(w, "invalid sort (created_at|id)", http.StatusBadRequest)
			return
		}
		q.Sort = s
	}

	// order whitelist
	if s := r.URL.Query().Get("order"); s != "" {
		s = strings.ToLower(s)
		if s != "asc" && s != "desc" {
			http.Error(w, "invalid order (asc|desc)", http.StatusBadRequest)
			return
		}
		q.Order = s
	}

	items, hasMore, err := h.rootsGetter.ListRoots(r.Context(), q)
	if err != nil {
		http.Error(w, "failed to get roots", http.StatusInternalServerError)
		return
	}

	resp := GetRootCommentsResponse{
		Items:   items,
		Limit:   q.Limit,
		Offset:  q.Offset,
		HasMore: hasMore,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(resp)
}
