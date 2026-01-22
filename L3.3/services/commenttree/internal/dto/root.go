package dto

// GetRootCommentsQuery описывает параметры запроса для получения корневых комментариев
type GetRootCommentsQuery struct {
	Limit  int
	Offset int
	Sort   string
	Order  string
}
