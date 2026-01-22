package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// DeleteCommentResponse describes response payload after deleting a comment.
type DeleteCommentResponse struct {
	DeletedID int64 `json:"deleted_id"`
}

// CommentDeleter deletes comments by id.
type CommentDeleter interface {
	Delete(ctx context.Context, id int64) error
}

// DeleteComment handles DELETE /comments/{id} and deletes a comment.
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, "missing path param: id", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.deleter.Delete(r.Context(), id); err != nil {
		http.Error(w, "failed to delete comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(DeleteCommentResponse{DeletedID: id})
}
