package handlers

import (
	"context"
	"encoding/json"
	"eventbooker/internal/model"
	"eventbooker/internal/repository"
	"eventbooker/internal/scheduler"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// EventHandler принимает HTTP запросы связанные с seat и events.
type EventHandler struct {
	db  repository.EventRepository
	s   *scheduler.CornScheduler
	log *slog.Logger
}

// NewEventHandler создает новый EventHandler.
func NewEventHandler(db repository.EventRepository, log *slog.Logger) *EventHandler {
	return &EventHandler{
		db:  db,
		s:   scheduler.New(15 * time.Second),
		log: log,
	}
}

// SetupScheduler запускает фоновый процесс проверки задач.
func (h *EventHandler) SetupScheduler() {
	h.s.Do(func(seatID int64) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		h.log.Info("checking payment status for seat", slog.Int64("seat_id", seatID))

		if err := h.db.CancelIfUnpaid(ctx, seatID); err != nil {
			h.log.Error("failed to automatically cancel booking",
				slog.Int64("seat_id", seatID),
				slog.Any("err", err),
			)
			return
		}

		h.log.Info("seat released via timeout (unpaid)", slog.Int64("seat_id", seatID))
	})
}

// ListEvents обрабатывает GET /events
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.db.ListEvents(r.Context())
	if err != nil {
		h.log.Error("failed to fetch events", slog.Any("err", err))
		http.Error(w, "Failed to list events", http.StatusInternalServerError)
		return
	}

	if events == nil {
		events = []model.Event{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(events); err != nil {
		h.log.Error("failed to encode events response", slog.Any("err", err))
	}
}

// CreateEvent обрабатывает POST /events
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var event model.Event

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.log.Warn("invalid request body for create event", slog.Any("err", err))
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() //nolint:errcheck

	if event.Name == "" {
		http.Error(w, "Event name is required", http.StatusBadRequest)
		return
	}
	if event.Capacity <= 0 {
		http.Error(w, "Capacity must be greater than 0", http.StatusBadRequest)
		return
	}

	eventID, err := h.db.CreateEvent(r.Context(), &event)
	if err != nil {
		h.log.Error("failed to create event in db", slog.Any("err", err))
		http.Error(w, "Failed to create event", http.StatusInternalServerError)
		return
	}

	h.log.Info("event created successfully", slog.Int64("event_id", eventID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      eventID,
		"message": "Event created successfully",
	}); err != nil {
		h.log.Error("failed to encode create event response", slog.Any("err", err))
	}
}

// GetEvent обрабатывает GET /events/{id}
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	event, err := h.db.GetEventByID(r.Context(), eventID)
	if err != nil {
		h.log.Warn("event not found", slog.Int64("event_id", eventID), slog.Any("err", err))
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	seats, err := h.db.ListSeatsForEvent(r.Context(), eventID)
	if err != nil {
		h.log.Error("failed to get seats for event", slog.Int64("event_id", eventID), slog.Any("err", err))
		http.Error(w, "Failed to get seats", http.StatusInternalServerError)
		return
	}

	freeSeatsCount := 0
	for _, seat := range seats {
		if seat.Status == model.SeatStatusFree {
			freeSeatsCount++
		}
	}

	response := map[string]interface{}{
		"event":            event,
		"free_seats_count": freeSeatsCount,
		"seats":            seats,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("failed to encode event detail response", slog.Any("err", err))
	}
}

// BookSpace обрабатывает POST /events/{id}/book
func (h *EventHandler) BookSpace(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	eventID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	var req struct {
		SeatID int64 `json:"seat_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() //nolint:errcheck

	if err := h.db.BookSeat(r.Context(), req.SeatID); err != nil {
		h.log.Warn("failed to book seat (conflict/unavailable)",
			slog.Int64("seat_id", req.SeatID),
			slog.Any("err", err),
		)
		http.Error(w, "Failed to book seat or seat unavailable", http.StatusConflict)
		return
	}

	h.s.AddItem(req.SeatID, 2*time.Minute)
	h.log.Info("seat booked, waiting for payment", slog.Int64("seat_id", req.SeatID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message":  "Seat booked successfully. Please pay within 2 minutes.",
		"event_id": eventID,
		"seat_id":  req.SeatID,
	}); err != nil {
		h.log.Error("failed to encode book response", slog.Any("err", err))
	}
}

// ConfirmBooking обрабатывает POST /events/{id}/confirm
func (h *EventHandler) ConfirmBooking(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	_, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	var req struct {
		SeatID int64 `json:"seat_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() //nolint:errcheck

	if err := h.db.PaySeat(r.Context(), req.SeatID); err != nil {
		h.log.Warn("failed to confirm booking",
			slog.Int64("seat_id", req.SeatID),
			slog.Any("err", err),
		)
		http.Error(w, "Failed to confirm booking or already confirmed/canceled", http.StatusBadRequest)
		return
	}

	h.log.Info("booking confirmed and paid", slog.Int64("seat_id", req.SeatID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Booking confirmed",
		"seat_id": req.SeatID,
	}); err != nil {
		h.log.Error("failed to encode confirm response", slog.Any("err", err))
	}
}
