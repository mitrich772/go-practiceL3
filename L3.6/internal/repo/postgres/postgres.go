// Package postgres реализует repo.Repo поверх PostgreSQL через библиотеку wbf/dbpg.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/wb-go/wbf/dbpg"

	"salestracker/internal/model"
	"salestracker/internal/repo"
)

// Repo реализует операции с финансовыми записями в PostgreSQL.
// Подключение идёт через *dbpg.DB — как в L3.5 (все запросы на db.Master).
type Repo struct {
	db *dbpg.DB
}

// New создаёт новый Repo.
func New(db *dbpg.DB) *Repo {
	return &Repo{db: db}
}

// Create вставляет новую запись и возвращает её с сгенерированными полями.
func (r *Repo) Create(ctx context.Context, item *model.Item) (model.Item, error) {
	const query = `
		INSERT INTO items (type, amount, category, note, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`

	var (
		id        int64
		createdAt sql.NullTime
	)
	err := r.db.Master.QueryRowContext(ctx, query,
		string(item.Type),
		item.Amount,
		item.Category,
		item.Note,
		item.OccurredAt,
	).Scan(&id, &createdAt)
	if err != nil {
		return model.Item{}, fmt.Errorf("insert item: %w", err)
	}

	out := *item
	out.ID = id
	if createdAt.Valid {
		out.CreatedAt = createdAt.Time
	}
	return out, nil
}

// Update обновляет существующую запись по ID.
func (r *Repo) Update(ctx context.Context, item *model.Item) (model.Item, error) {
	const query = `
		UPDATE items
		SET type        = $1,
		    amount      = $2,
		    category    = $3,
		    note        = $4,
		    occurred_at = $5
		WHERE id = $6
		RETURNING id, type, amount, category, note, occurred_at, created_at
	`

	var out model.Item
	var typ string
	err := r.db.Master.QueryRowContext(ctx, query,
		string(item.Type),
		item.Amount,
		item.Category,
		item.Note,
		item.OccurredAt,
		item.ID,
	).Scan(&out.ID, &typ, &out.Amount, &out.Category, &out.Note, &out.OccurredAt, &out.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Item{}, repo.ErrNotFound
		}
		return model.Item{}, fmt.Errorf("update item: %w", err)
	}
	out.Type = model.ItemType(typ)
	return out, nil
}

// Delete удаляет запись по ID. Если записи нет — возвращает repo.ErrNotFound.
func (r *Repo) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM items WHERE id = $1`

	res, err := r.db.Master.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return repo.ErrNotFound
	}
	return nil
}

// GetByID возвращает запись по ID.
func (r *Repo) GetByID(ctx context.Context, id int64) (model.Item, error) {
	const query = `
		SELECT id, type, amount, category, note, occurred_at, created_at
		FROM items
		WHERE id = $1
	`

	var out model.Item
	var typ string
	err := r.db.Master.QueryRowContext(ctx, query, id).Scan(
		&out.ID, &typ, &out.Amount, &out.Category, &out.Note, &out.OccurredAt, &out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Item{}, repo.ErrNotFound
		}
		return model.Item{}, fmt.Errorf("get item: %w", err)
	}
	out.Type = model.ItemType(typ)
	return out, nil
}

// List возвращает страницу записей по фильтру + флаг "есть ли следующая страница".
//
// Сортировка ограничена whitelist'ом, поэтому safe-fmt в ORDER BY не приводит к SQL-инъекции.
func (r *Repo) List(ctx context.Context, f model.ItemFilter) ([]model.Item, bool, error) {
	conds, args := buildWhere(f)

	sortCol := whitelistSort(f.Sort)
	order := whitelistOrder(f.Order)

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	limitPlus := limit + 1

	args = append(args, limitPlus, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(args)-1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(args))

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, type, amount, category, note, occurred_at, created_at
		FROM items
		%s
		ORDER BY %s %s, id %s
		LIMIT %s OFFSET %s
	`, where, sortCol, order, order, limitPlaceholder, offsetPlaceholder)

	rows, err := r.db.Master.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	items := make([]model.Item, 0, limitPlus)
	for rows.Next() {
		var it model.Item
		var typ string
		if err := rows.Scan(&it.ID, &typ, &it.Amount, &it.Category, &it.Note, &it.OccurredAt, &it.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan item: %w", err)
		}
		it.Type = model.ItemType(typ)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("rows: %w", err)
	}

	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}
	return items, hasMore, nil
}

// Analytics считает агрегаты строго по ТЗ: count, sum, avg, median, p90.
// Медиана и P90 — через percentile_cont (SQL).
func (r *Repo) Analytics(ctx context.Context, f model.ItemFilter) (model.Analytics, error) {
	conds, args := buildWhere(f)
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	aggQuery := fmt.Sprintf(`
		SELECT
			COUNT(*)                                                         AS count,
			COALESCE(SUM(amount), 0)                                         AS sum,
			COALESCE(AVG(amount), 0)                                         AS avg,
			COALESCE(percentile_cont(0.5) WITHIN GROUP (ORDER BY amount), 0) AS median,
			COALESCE(percentile_cont(0.9) WITHIN GROUP (ORDER BY amount), 0) AS p90
		FROM items
		%s
	`, where)

	var a model.Analytics
	a.From = f.From
	a.To = f.To
	err := r.db.Master.QueryRowContext(ctx, aggQuery, args...).Scan(
		&a.Count, &a.Sum, &a.Avg, &a.Median, &a.Percentile90,
	)
	if err != nil {
		return model.Analytics{}, fmt.Errorf("aggregate: %w", err)
	}

	return a, nil
}

// buildWhere формирует список условий и параметров запроса по фильтру.
// Все значения уходят как параметры $N — никакой интерполяции пользовательского ввода.
func buildWhere(f model.ItemFilter) ([]string, []any) {
	conds := make([]string, 0, 4)
	args := make([]any, 0, 4)

	if f.From != nil {
		args = append(args, *f.From)
		conds = append(conds, fmt.Sprintf("occurred_at >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		conds = append(conds, fmt.Sprintf("occurred_at <= $%d", len(args)))
	}
	if f.Type != "" {
		args = append(args, string(f.Type))
		conds = append(conds, fmt.Sprintf("type = $%d", len(args)))
	}
	if f.Category != "" {
		args = append(args, f.Category)
		conds = append(conds, fmt.Sprintf("category = $%d", len(args)))
	}
	return conds, args
}

func whitelistSort(s string) string {
	switch s {
	case "amount", "id":
		return s
	default:
		return "occurred_at"
	}
}

func whitelistOrder(s string) string {
	if strings.ToLower(s) == "asc" {
		return "ASC"
	}
	return "DESC"
}
