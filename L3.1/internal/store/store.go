package store

import (
	"L3.1/internal/types"
)

// Store — абстракция хранилища уведомлений.
type Store interface {
	Save(id string, ntf types.Notification) error
	Get(id string) (*types.Notification, error)
	Cancel(id string) error
	Update(id string, ntf types.Notification) error
}
