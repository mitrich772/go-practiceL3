package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type DeleteCommentResponse struct {
	DeletedID int64 `json:"deleted_id"`
}

type CommentDeleter interface {
	Delete(ctx context.Context, id int64) error
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	parentIDstr := chi.URLParam(r, "id")
	if parentIDstr == "" {
		http.Error(w, "missing query param: parent", http.StatusBadRequest)
		return
	}

	parentID, err := strconv.ParseInt(parentIDstr, 10, 64)
	if err != nil || parentID <= 0 {
		http.Error(w, "invalid parent id", http.StatusBadRequest)
		return
	}

	if err := h.deleter.Delete(r.Context(), parentID); err != nil {
		http.Error(w, "failed to delete comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(DeleteCommentResponse{DeletedID: parentID})
}
