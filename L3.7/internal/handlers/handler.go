package handlers

import (
	"log/slog"
	"time"
)

// Handler группирует HTTP-хендлеры warehousecontrol и их зависимости.
type Handler struct {
	creator ItemCreator
	updater ItemUpdater
	deleter ItemDeleter
	lister  ItemsLister
	getter  ItemGetter
	history HistoryGetter
	users   UsersLister

	jwtSecret []byte
	tokenTTL  time.Duration
	log       *slog.Logger
}

// New создаёт *Handler. Все зависимости передаются явно через verb-er интерфейсы.
func New(
	creator ItemCreator,
	updater ItemUpdater,
	deleter ItemDeleter,
	lister ItemsLister,
	getter ItemGetter,
	history HistoryGetter,
	users UsersLister,
	jwtSecret string,
	tokenTTL time.Duration,
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
		history:   history,
		users:     users,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
		log:       log.With(slog.String("component", "handlers")),
	}
}
