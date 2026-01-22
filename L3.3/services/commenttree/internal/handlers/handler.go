package handlers

// Handler groups HTTP handlers and their dependencies.
type Handler struct {
	creator     CommentCreator
	getter      CommentGetter
	deleter     CommentDeleter
	rootsGetter RootCommentsGetter
	searcher    CommentSearcher
}

// New builds a Handler with all required dependencies.
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
