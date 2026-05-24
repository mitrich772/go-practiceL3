// Package model содержит доменные сущности WarehouseControl.
package model

import (
	"encoding/json"
	"time"
)

// Role описывает роль пользователя в системе.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleViewer  Role = "viewer"
)

func (r Role) Valid() bool {
	return r == RoleAdmin || r == RoleManager || r == RoleViewer
}

func (r Role) CanWrite() bool {
	return r == RoleAdmin || r == RoleManager
}

func (r Role) CanDelete() bool {
	return r == RoleAdmin
}

func (r Role) CanViewHistory() bool {
	return r == RoleAdmin
}

// User — пользователь демо-системы.
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// AuthUser — пользователь, извлечённый из JWT.
type AuthUser struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
}

// Item — товар на складе.
type Item struct {
	ID          int64     `json:"id"          db:"id"`
	Name        string    `json:"name"        db:"name"`
	SKU         string    `json:"sku"         db:"sku"`
	Quantity    int       `json:"quantity"    db:"quantity"`
	Location    string    `json:"location"    db:"location"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at"  db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"  db:"updated_at"`
}

// ItemFilter описывает параметры выборки товаров.
type ItemFilter struct {
	Search string
	Limit  int
	Offset int
	Sort   string // id | name | sku | quantity | updated_at
	Order  string // asc | desc
}

// HistoryEntry — запись аудита из item_history, которую создаёт DB-trigger.
type HistoryEntry struct {
	ID        int64           `json:"id"`
	ItemID    *int64          `json:"item_id,omitempty"`
	Action    string          `json:"action"`
	Actor     string          `json:"actor"`
	ActorRole string          `json:"actor_role"`
	OldData   json.RawMessage `json:"old_data,omitempty"`
	NewData   json.RawMessage `json:"new_data,omitempty"`
	ChangedAt time.Time       `json:"changed_at"`
}
