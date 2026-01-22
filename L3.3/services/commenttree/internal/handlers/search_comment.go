package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"commenttree/internal/dto"
)

// CommentSearcher searches comments by query.
type CommentSearcher interface {
	Search(ctx context.Context, q dto.SearchCommentsQuery) ([]dto.CommentNode, bool, error)
}

// SearchComments handles GET /search and returns matched comments.
func (h *Handler) SearchComments(w http.ResponseWriter, r *http.Request) {
	// Default query settings.
	q := dto.SearchCommentsQuery{
		Q:      strings.TrimSpace(r.URL.Query().Get("q")),
		Limit:  20,
		Offset: 0,
		Sort:   "rank",
		Order:  "desc",
	}

	if q.Q == "" {
		http.Error(w, "missing query param: q", http.StatusBadRequest)
		return
	}

	if s := r.URL.Query().Get("limit"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 || v > 100 {
			http.Error(w, "invalid limit (1..100)", http.StatusBadRequest)
			return
		}
		q.Limit = v
	}

	if s := r.URL.Query().Get("offset"); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			http.Error(w, "invalid offset (>=0)", http.StatusBadRequest)
			return
		}
		q.Offset = v
	}

	// Whitelist sort fields.
	if s := r.URL.Query().Get("sort"); s != "" {
		s = strings.ToLower(s)
		if s != "rank" && s != "created_at" && s != "id" {
			http.Error(w, "invalid sort (rank|created_at|id)", http.StatusBadRequest)
			return
		}
		q.Sort = s
	}

	// Whitelist order values.
	if s := r.URL.Query().Get("order"); s != "" {
		s = strings.ToLower(s)
		if s != "asc" && s != "desc" {
			http.Error(w, "invalid order (asc|desc)", http.StatusBadRequest)
			return
		}
		q.Order = s
	}

	items, hasMore, err := h.searcher.Search(r.Context(), q)
	if err != nil {
		http.Error(w, "failed to search comments", http.StatusInternalServerError)
		return
	}

	resp := dto.SearchCommentsResponse{
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
