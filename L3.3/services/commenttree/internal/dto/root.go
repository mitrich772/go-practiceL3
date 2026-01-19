package dto

type GetRootCommentsQuery struct {
	Limit  int
	Offset int
	Sort   string
	Order  string
}
