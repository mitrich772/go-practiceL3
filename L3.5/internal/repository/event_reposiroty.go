package repository

import (
	"context"
	"eventbooker/internal/model"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, event *model.Event) (int64, error)
	GetEventByID(ctx context.Context, eventID int64) (*model.Event, error)
	ListEvents(ctx context.Context) ([]model.Event, error)
	UpdateEvent(ctx context.Context, event *model.Event) error
	DeleteEvent(ctx context.Context, eventID int64) error

	ListSeatsForEvent(ctx context.Context, eventID int64) ([]model.Seat, error)
	BookSeat(ctx context.Context, seatID int64) error
	PaySeat(ctx context.Context, seatID int64) error
	CancelReservation(ctx context.Context, seatID int64) error
	CancelIfUnpaid(ctx context.Context, seatID int64) error
}
