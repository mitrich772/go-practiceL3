package dto

import "time"

// Структура для овтета созданного коммента
type Comment struct {
	ID        int64
	ParentID  *int64
	Body      string
	CreatedAt time.Time
}

type CommentNode struct {
	ID            int64         `json:"id"`
	ParentID      *int64        `json:"parent_id"`
	Body          string        `json:"body"`
	CreatedAt     time.Time     `json:"created_at"`
	ChildrenCount int           `json:"children_count"`
	Children      []CommentNode `json:"children"`

	Rank float64 `json:"rank,omitempty"` //для поиска
}
