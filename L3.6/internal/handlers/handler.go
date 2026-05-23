package handlers

import "log/slog"

// Handler группирует HTTP-хендлеры salestracker и их зависимости.
type Handler struct {
	creator   ItemCreator
	updater   ItemUpdater
	deleter   ItemDeleter
	lister    ItemsLister
	getter    ItemGetter
	analytics AnalyticsGetter

	log *slog.Logger
}

// New создаёт *Handler. Все зависимости передаются явно через verb-er интерфейсы.
func New(
	creator ItemCreator,
	updater ItemUpdater,
	deleter ItemDeleter,
	lister ItemsLister,
	getter ItemGetter,
	analytics AnalyticsGetter,
	log *slog.Logger,
) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{
		creator:   creator,
		updater:   updater,
		deleter:   deleter,
		lister:    lister,
		getter:    getter,
		analytics: analytics,
		log:       log.With(slog.String("component", "handlers")),
	}
}
