package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"commenttree/internal/dto"
	stErr "commenttree/internal/store/postgres/errors"
)

// GetCommentsResponse describes response payload for subtree request.
type GetCommentsResponse struct {
	Tree dto.CommentNode `json:"tree"`
}

// CommentGetter returns a comment subtree.
type CommentGetter interface {
	GetSubtree(ctx context.Context, rootID int64, maxDepth int) (dto.CommentNode, error)
}

// GetComments handles GET /comments?parent={id}&depth={maxDepth} and returns a subtree.
func (h *Handler) GetComments(w http.ResponseWriter, r *http.Request) {
	parentIDstr := r.URL.Query().Get("parent")
	if parentIDstr == "" {
		http.Error(w, "missing query param: parent", http.StatusBadRequest)
		return
	}

	parentID, err := strconv.ParseInt(parentIDstr, 10, 64)
	if err != nil || parentID <= 0 {
		http.Error(w, "invalid parent id", http.StatusBadRequest)
		return
	}

	depthStr := r.URL.Query().Get("depth")
	maxDepth := -1
	if depthStr != "" {
		d, err := strconv.Atoi(depthStr)
		if err != nil || d < -1 {
			http.Error(w, "invalid depth", http.StatusBadRequest)
			return
		}
		maxDepth = d
	}

	tree, err := h.getter.GetSubtree(r.Context(), parentID, maxDepth)
	if err != nil {
		if errors.Is(err, stErr.ErrNotFound) {
			http.Error(w, "comment not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get comments", http.StatusInternalServerError)
		return
	}

	resp := GetCommentsResponse{Tree: tree}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(resp); err != nil {
		http.Error(w, "failed to encode json", http.StatusInternalServerError)
		return
	}
}
