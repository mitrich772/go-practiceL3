package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"commenttree/internal/dto"
)

// CreateCommentRequest describes request payload for creating a comment.
type CreateCommentRequest struct {
	// ParentID is null for a root comment.
	ParentID *int64 `json:"parent_id"`
	Body     string `json:"body"`
}

// CreateCommentResponse describes response payload after creating a comment.
type CreateCommentResponse struct {
	CreatedComment dto.Comment `json:"created_comment"`
}

// CommentCreator creates comments in storage.
type CommentCreator interface {
	Create(ctx context.Context, parentID *int64, body string) (dto.Comment, error)
}

// CreateComment handles POST /comments and creates a new comment.
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var req CreateCommentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json body", http.StatusBadRequest)
		return
	}

	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}

	createdComment, err := h.creator.Create(r.Context(), req.ParentID, req.Body)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "failed to create comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(CreateCommentResponse{CreatedComment: createdComment})
}
