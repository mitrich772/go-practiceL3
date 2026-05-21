package postgres

import (
	"context"
	"eventbooker/internal/model"
	"fmt"

	"github.com/wb-go/wbf/dbpg"
)

// PostgresRepository использует библиотеку wbf для работы с БД
type PostgresRepository struct {
	db *dbpg.DB
}

// NewPostgresRepository — конструктор репозитория
func NewPostgresRepository(db *dbpg.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

// CreateEvent создаёт мероприятие и автоматически генерирует N мест.
// Всё выполняется в рамках одной транзакции.
func (r *PostgresRepository) CreateEvent(ctx context.Context, event *model.Event) (int64, error) {
	tx, err := r.db.Master.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is no-op

	var newID int64
	query := `
		INSERT INTO events (name, description, event_date, capacity) 
		VALUES ($1, $2, $3, $4) 
		RETURNING id
	`
	err = tx.QueryRowContext(ctx, query, event.Name, event.Description, event.EventDate, event.Capacity).Scan(&newID)
	if err != nil {
		return 0, fmt.Errorf("insert event: %w", err)
	}

	// Генерируем N мест для мероприятия
	seatQuery := `
		INSERT INTO seats (event_id, seat_number, status) 
		VALUES ($1, $2, 'free')
	`
	for i := 1; i <= event.Capacity; i++ {
		seatNumber := fmt.Sprintf("%d", i)
		_, err = tx.ExecContext(ctx, seatQuery, newID, seatNumber)
		if err != nil {
			return 0, fmt.Errorf("insert seat %d: %w", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return newID, nil
}

// GetEventByID возвращает мероприятие по ID.
func (r *PostgresRepository) GetEventByID(ctx context.Context, eventID int64) (*model.Event, error) {
	var event model.Event
	query := `
		SELECT id, name, description, event_date, capacity, created_at 
		FROM events 
		WHERE id = $1
	`
	err := r.db.Master.QueryRowContext(ctx, query, eventID).Scan(
		&event.ID, &event.Name, &event.Description, &event.EventDate, &event.Capacity, &event.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// ListEvents возвращает список всех мероприятий.
func (r *PostgresRepository) ListEvents(ctx context.Context) ([]model.Event, error) {
	query := `
		SELECT id, name, description, event_date, capacity, created_at 
		FROM events 
		ORDER BY event_date ASC
	`
	rows, err := r.db.Master.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var events []model.Event
	for rows.Next() {
		var event model.Event
		if err := rows.Scan(&event.ID, &event.Name, &event.Description, &event.EventDate, &event.Capacity, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// UpdateEvent обновляет мероприятие.
func (r *PostgresRepository) UpdateEvent(ctx context.Context, event *model.Event) error {
	query := `
		UPDATE events 
		SET name = $1, description = $2, event_date = $3 
		WHERE id = $4
	`
	_, err := r.db.Master.ExecContext(ctx, query, event.Name, event.Description, event.EventDate, event.ID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteEvent удаляет мероприятие.
func (r *PostgresRepository) DeleteEvent(ctx context.Context, eventID int64) error {
	query := `
		DELETE FROM events 
		WHERE id = $1
	`
	_, err := r.db.Master.ExecContext(ctx, query, eventID)
	if err != nil {
		return err
	}
	return nil
}

// ListSeatsForEvent возвращает список всех мест для мероприятия.
func (r *PostgresRepository) ListSeatsForEvent(ctx context.Context, eventID int64) ([]model.Seat, error) {
	query := `
		SELECT id, event_id, seat_number, status, booked_at, updated_at 
		FROM seats 
		WHERE event_id = $1
		ORDER BY id ASC
	`
	rows, err := r.db.Master.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	var seats []model.Seat
	for rows.Next() {
		var seat model.Seat
		if err := rows.Scan(&seat.ID, &seat.EventID, &seat.SeatNumber, &seat.Status, &seat.BookedAt, &seat.UpdatedAt); err != nil {
			return nil, err
		}
		seats = append(seats, seat)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return seats, nil
}

// BookSeat бронирует свободное место. Использует транзакцию для атомарности.
// Обновляет статус free → reserved и устанавливает booked_at.
func (r *PostgresRepository) BookSeat(ctx context.Context, seatID int64) error {
	tx, err := r.db.Master.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is no-op

	query := `
		UPDATE seats 
		SET status = $1, booked_at = NOW(), updated_at = NOW() 
		WHERE id = $2 AND status = $3
	`
	res, err := tx.ExecContext(ctx, query, model.SeatStatusReserved, seatID, model.SeatStatusFree)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("seat not available or already booked")
	}

	return tx.Commit()
}

// PaySeat подтверждает оплату забронированного места.
// Обновляет статус reserved на confirmed. Использует транзакцию.
func (r *PostgresRepository) PaySeat(ctx context.Context, seatID int64) error {
	tx, err := r.db.Master.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is no-op

	query := `
		UPDATE seats 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND status = $3
	`
	res, err := tx.ExecContext(ctx, query, model.SeatStatusConfirmed, seatID, model.SeatStatusReserved)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("seat not reserved or already confirmed/canceled")
	}

	return tx.Commit()
}

// CancelReservation отменяет бронирование, возвращает место в статус free.
func (r *PostgresRepository) CancelReservation(ctx context.Context, seatID int64) error {
	query := `
		UPDATE seats 
		SET status = $1, booked_at = NULL, updated_at = NOW() 
		WHERE id = $2 AND status = $3
	`
	res, err := r.db.Master.ExecContext(ctx, query, model.SeatStatusFree, seatID, model.SeatStatusReserved)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("seat not reserved")
	}
	return nil
}

// CancelIfUnpaid автоматически отменяет бронь, если она до сих пор не оплачена.
// Выполняет UPDATE только если текущий статус = 'reserved'.
// Если статус уже 'confirmed', запрос ничего не изменит (0 rows affected).
// Используется фоновым планировщиком.
func (r *PostgresRepository) CancelIfUnpaid(ctx context.Context, seatID int64) error {
	tx, err := r.db.Master.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback after commit is no-op

	query := `
		UPDATE seats 
		SET status = 'free', booked_at = NULL, updated_at = NOW() 
		WHERE id = $1 AND status = 'reserved'
	`
	_, err = tx.ExecContext(ctx, query, seatID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
