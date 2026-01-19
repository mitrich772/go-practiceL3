package handlers

type Handler struct {
	creator     CommentCreator
	getter      CommentGetter
	deleter     CommentDeleter
	rootsGetter RootCommentsGetter
	searcher    CommentSearcher
}

func New(
	creator CommentCreator,
	getter CommentGetter,
	deleter CommentDeleter,
	rootsGetter RootCommentsGetter,
	searcher CommentSearcher,
) *Handler {
	return &Handler{
		creator:     creator,
		getter:      getter,
		deleter:     deleter,
		rootsGetter: rootsGetter,
		searcher:    searcher,
	}
}
