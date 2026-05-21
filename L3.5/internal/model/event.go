package model

import "time"

// Event описывает сущность мероприятия в системе.
type Event struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	EventDate   time.Time `json:"event_date" db:"event_date"`
	Capacity    int       `json:"capacity" db:"capacity"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// SeatStatus представляет текущий статус места.
type SeatStatus string

const (
	SeatStatusFree      SeatStatus = "free"
	SeatStatusReserved  SeatStatus = "reserved"
	SeatStatusPaid      SeatStatus = "paid"
	SeatStatusConfirmed SeatStatus = "confirmed"
)

// Seat описывает конкретное место на мероприятии с указанием его статуса.
type Seat struct {
	ID         int64      `json:"id" db:"id"`
	EventID    int64      `json:"event_id" db:"event_id"`
	SeatNumber string     `json:"seat_number" db:"seat_number"`
	Status     SeatStatus `json:"status" db:"status"`
	BookedAt   *time.Time `json:"booked_at,omitempty" db:"booked_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// EventFilter используется для спецификации параметров фильтрации и пагинации списка мероприятий.
type EventFilter struct {
	Query  string `json:"query"` // для поиска по имени/описанию
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
}
