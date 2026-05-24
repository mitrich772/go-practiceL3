// Package postgres реализует repo.Repo поверх PostgreSQL через библиотеку wbf/dbpg.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/wb-go/wbf/dbpg"

	"warehousecontrol/internal/model"
	"warehousecontrol/internal/repo"
)

// Repo реализует операции склада в PostgreSQL.
type Repo struct {
	db *dbpg.DB
}

// New создаёт новый Repo.
func New(db *dbpg.DB) *Repo {
	return &Repo{db: db}
}

// Create вставляет новый товар. Триггер БД сам запишет историю с app.actor/app.role.
func (r *Repo) Create(ctx context.Context, item *model.Item, actor model.AuthUser) (model.Item, error) {
	const query = `
		INSERT INTO items (name, sku, quantity, location, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, sku, quantity, location, description, created_at, updated_at
	`

	var out model.Item
	err := r.withActor(ctx, actor, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query,
			item.Name,
			item.SKU,
			item.Quantity,
			item.Location,
			item.Description,
		).Scan(&out.ID, &out.Name, &out.SKU, &out.Quantity, &out.Location, &out.Description, &out.CreatedAt, &out.UpdatedAt)
	})
	if err != nil {
		return model.Item{}, fmt.Errorf("insert item: %w", err)
	}
	return out, nil
}

// Update обновляет существующий товар по ID.
func (r *Repo) Update(ctx context.Context, item *model.Item, actor model.AuthUser) (model.Item, error) {
	const query = `
		UPDATE items
		SET name = $1,
		    sku = $2,
		    quantity = $3,
		    location = $4,
		    description = $5
		WHERE id = $6
		RETURNING id, name, sku, quantity, location, description, created_at, updated_at
	`

	var out model.Item
	err := r.withActor(ctx, actor, func(tx *sql.Tx) error {
		return tx.QueryRowContext(ctx, query,
			item.Name,
			item.SKU,
			item.Quantity,
			item.Location,
			item.Description,
			item.ID,
		).Scan(&out.ID, &out.Name, &out.SKU, &out.Quantity, &out.Location, &out.Description, &out.CreatedAt, &out.UpdatedAt)
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Item{}, repo.ErrNotFound
		}
		return model.Item{}, fmt.Errorf("update item: %w", err)
	}
	return out, nil
}

// Delete удаляет товар по ID. Если записи нет — возвращает repo.ErrNotFound.
func (r *Repo) Delete(ctx context.Context, id int64, actor model.AuthUser) error {
	const query = `DELETE FROM items WHERE id = $1`

	err := r.withActor(ctx, actor, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return repo.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return repo.ErrNotFound
		}
		return fmt.Errorf("delete item: %w", err)
	}
	return nil
}

// GetByID возвращает товар по ID.
func (r *Repo) GetByID(ctx context.Context, id int64) (model.Item, error) {
	const query = `
		SELECT id, name, sku, quantity, location, description, created_at, updated_at
		FROM items
		WHERE id = $1
	`

	var out model.Item
	err := r.db.Master.QueryRowContext(ctx, query, id).Scan(
		&out.ID, &out.Name, &out.SKU, &out.Quantity, &out.Location, &out.Description, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Item{}, repo.ErrNotFound
		}
		return model.Item{}, fmt.Errorf("get item: %w", err)
	}
	return out, nil
}

// List возвращает страницу товаров по фильтру + флаг следующей страницы.
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
		SELECT id, name, sku, quantity, location, description, created_at, updated_at
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
		if err := rows.Scan(&it.ID, &it.Name, &it.SKU, &it.Quantity, &it.Location, &it.Description, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, false, fmt.Errorf("scan item: %w", err)
		}
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

// History возвращает историю изменений по item_id.
func (r *Repo) History(ctx context.Context, itemID int64) ([]model.HistoryEntry, error) {
	const query = `
		SELECT id, item_id, action, actor, actor_role, old_data, new_data, changed_at
		FROM item_history
		WHERE item_id = $1
		ORDER BY changed_at DESC, id DESC
	`

	rows, err := r.db.Master.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("history: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	entries := make([]model.HistoryEntry, 0)
	for rows.Next() {
		var entry model.HistoryEntry
		var item sql.NullInt64
		var oldData, newData []byte
		if err := rows.Scan(&entry.ID, &item, &entry.Action, &entry.Actor, &entry.ActorRole, &oldData, &newData, &entry.ChangedAt); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		if item.Valid {
			entry.ItemID = &item.Int64
		}
		entry.OldData = json.RawMessage(oldData)
		entry.NewData = json.RawMessage(newData)
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("history rows: %w", err)
	}
	return entries, nil
}

// ListUsers возвращает демо-пользователей для UI.
func (r *Repo) ListUsers(ctx context.Context) ([]model.User, error) {
	const query = `SELECT id, username, role FROM users ORDER BY id`
	rows, err := r.db.Master.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	users := make([]model.User, 0)
	for rows.Next() {
		var u model.User
		var role string
		if err := rows.Scan(&u.ID, &u.Username, &role); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		u.Role = model.Role(role)
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user rows: %w", err)
	}
	return users, nil
}

func (r *Repo) withActor(ctx context.Context, actor model.AuthUser, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.Master.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.actor', $1, true), set_config('app.role', $2, true)`, actor.Username, string(actor.Role)); err != nil {
		return fmt.Errorf("set actor: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func buildWhere(f model.ItemFilter) ([]string, []any) {
	conds := make([]string, 0, 1)
	args := make([]any, 0, 1)

	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		conds = append(conds, fmt.Sprintf("(lower(name) LIKE $%d OR lower(sku) LIKE $%d OR lower(location) LIKE $%d)", len(args), len(args), len(args)))
	}
	return conds, args
}

func whitelistSort(s string) string {
	switch s {
	case "name", "sku", "quantity", "id":
		return s
	default:
		return "updated_at"
	}
}

func whitelistOrder(s string) string {
	if strings.ToLower(s) == "asc" {
		return "ASC"
	}
	return "DESC"
}
