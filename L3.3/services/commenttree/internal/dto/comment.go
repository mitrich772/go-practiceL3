package dto

import "time"

// Comment представляет плоский комментарий (как строка из БД), например для ответа POST /comments.
type Comment struct {
	ID        int64     `json:"id"`
	ParentID  *int64    `json:"parent_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentNode представляет комментарий в виде узла дерева.
type CommentNode struct {
	ID            int64         `json:"id"`
	ParentID      *int64        `json:"parent_id"`
	Body          string        `json:"body"`
	CreatedAt     time.Time     `json:"created_at"`
	ChildrenCount int           `json:"children_count"`
	Children      []CommentNode `json:"children"`

	// Rank используется только в результате полнотекстового поиска.
	Rank float64 `json:"rank,omitempty"`
}
