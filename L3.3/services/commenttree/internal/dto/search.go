package dto

// SearchCommentsQuery описывает параметры полнотекстового поиска комментариев
type SearchCommentsQuery struct {
	Q      string
	Limit  int
	Offset int
	Sort   string // rank|created_at|id
	Order  string // asc|desc
}

// SearchCommentsResponse описывает ответ эндпоинта поиска комментариев.
type SearchCommentsResponse struct {
	Items   []CommentNode `json:"items"`
	Limit   int           `json:"limit"`
	Offset  int           `json:"offset"`
	HasMore bool          `json:"has_more"`
}
