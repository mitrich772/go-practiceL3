package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"commenttree/internal/dto"
)

type CreateCommentRequest struct {
	ParentID *int64 `json:"parent_id"` // null -> корневой
	Body     string `json:"body"`
}

type CreateCommentResponse struct {
	Created_comment dto.Comment `json:"created_comment"`
}

type CommentCreator interface {
	Create(ctx context.Context, parentID *int64, body string) (dto.Comment, error)
}

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

	created_comment, err := h.creator.Create(r.Context(), req.ParentID, req.Body)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "failed to create comment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(CreateCommentResponse{Created_comment: created_comment})
}
