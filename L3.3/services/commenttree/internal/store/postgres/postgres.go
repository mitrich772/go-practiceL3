package postgres

import (
	"commenttree/internal/dto"
	trUtil "commenttree/internal/store/utils"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type StorePG struct {
	db *sql.DB
}

// New создаёт StorePG.
func New(db *sql.DB) *StorePG {
	return &StorePG{db: db}
}

func (s *StorePG) Create(ctx context.Context, parentID *int64, body string) (dto.Comment, error) {
	const query = `
		INSERT INTO comments (parent_id, body)
		VALUES ($1, $2)
		RETURNING id
	`

	var CreatedID int64
	err := s.db.QueryRowContext(ctx, query, parentID, body).Scan(&CreatedID)
	if err != nil {
		return dto.Comment{}, err
	}
	created := dto.Comment{
		ID:        CreatedID,
		ParentID:  parentID,
		Body:      body,
		CreatedAt: time.Now(),
	}
	return created, nil
}

func (s *StorePG) GetSubtree(ctx context.Context, rootID int64, maxDepth int) (dto.CommentNode, error) {
	if maxDepth < 0 {
		maxDepth = 1000
	}
	const query = `
	WITH RECURSIVE tree AS (
	SELECT
		c.id,
		c.parent_id,
		c.body,
		c.created_at,
		0 AS depth,
		ARRAY[c.id] AS path
	FROM comments c
	WHERE c.id = $1

	UNION ALL

	SELECT
		ch.id,
		ch.parent_id,
		ch.body,
		ch.created_at,
		t.depth + 1,
		t.path || ch.id
	FROM comments ch
	JOIN tree t ON ch.parent_id = t.id
	WHERE t.depth < $2
	)
	SELECT id, parent_id, body, created_at, depth
	FROM tree
	ORDER BY path;
	`

	rows, err := s.db.QueryContext(ctx, query, rootID, maxDepth)
	if err != nil {
		return dto.CommentNode{}, err
	}
	defer rows.Close()

	flat := make([]dto.CommentNode, 0, 32)

	for rows.Next() {
		var (
			id      int64
			parent  sql.NullInt64
			body    string
			created time.Time
			depth   int
		)

		if err := rows.Scan(&id, &parent, &body, &created, &depth); err != nil {
			return dto.CommentNode{}, err
		}

		var pid *int64
		if parent.Valid {
			v := parent.Int64
			pid = &v
		}

		flat = append(flat, dto.CommentNode{
			ID:        id,
			ParentID:  pid,
			Body:      body,
			CreatedAt: created,
			Children:  []dto.CommentNode{},
		})
	}

	if err := rows.Err(); err != nil {
		return dto.CommentNode{}, err
	}

	tree, err := trUtil.BuildCommentTree(rootID, flat)
	if err != nil {
		return dto.CommentNode{}, err
	}

	return tree, nil
}

// сделать возврат id удаленного ?
func (s *StorePG) Delete(ctx context.Context, id int64) error {
	const query = `
		DELETE FROM comments WHERE id = $1;
	`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *StorePG) ListRoots(ctx context.Context, q dto.GetRootCommentsQuery) ([]dto.CommentNode, bool, error) {
	orderBy := q.Sort + " " + q.Order
	limitPlus := q.Limit + 1

	query := `
		SELECT
		c.id,
		c.parent_id,
		c.body,
		c.created_at,
		(SELECT COUNT(*) FROM comments ch WHERE ch.parent_id = c.id) AS children_count
		FROM comments c
		WHERE c.parent_id IS NULL
		ORDER BY ` + orderBy + `
		LIMIT $1 OFFSET $2;
	`

	rows, err := s.db.QueryContext(ctx, query, limitPlus, q.Offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]dto.CommentNode, 0, q.Limit+1)

	for rows.Next() {
		var it dto.CommentNode
		if err := rows.Scan(&it.ID, &it.ParentID, &it.Body, &it.CreatedAt, &it.ChildrenCount); err != nil {
			return nil, false, err
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(items) > q.Limit {
		hasMore = true
		items = items[:q.Limit]
	}

	return items, hasMore, nil
}

func (s *StorePG) Search(ctx context.Context, q dto.SearchCommentsQuery) ([]dto.CommentNode, bool, error) {
	// whitelist для ORDER BY (чтобы не было SQL injection)
	sortCol := "rank"
	switch strings.ToLower(q.Sort) {
	case "rank":
		sortCol = "rank"
	case "created_at":
		sortCol = "created_at"
	case "id":
		sortCol = "id"
	}

	order := "desc"
	switch strings.ToLower(q.Order) {
	case "asc":
		order = "asc"
	case "desc":
		order = "desc"
	}

	limitPlus := q.Limit + 1
	orderBy := fmt.Sprintf("%s %s", sortCol, order)

	query := `
SELECT
  c.id,
  c.parent_id,
  c.body,
  c.created_at,
  (SELECT COUNT(*) FROM comments ch WHERE ch.parent_id = c.id) AS children_count,
  ts_rank_cd(to_tsvector('russian', c.body), qq) AS rank
FROM comments c,
     websearch_to_tsquery('russian', $1) qq
WHERE to_tsvector('russian', c.body) @@ qq
ORDER BY ` + orderBy + `, c.created_at DESC
LIMIT $2 OFFSET $3;
`

	rows, err := s.db.QueryContext(ctx, query, q.Q, limitPlus, q.Offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]dto.CommentNode, 0, q.Limit+1)

	for rows.Next() {
		var it dto.CommentNode
		var pid sql.NullInt64

		if err := rows.Scan(
			&it.ID,
			&pid,
			&it.Body,
			&it.CreatedAt,
			&it.ChildrenCount,
			&it.Rank,
		); err != nil {
			return nil, false, err
		}

		if pid.Valid {
			v := pid.Int64
			it.ParentID = &v
		} else {
			it.ParentID = nil
		}

		it.Children = []dto.CommentNode{} // на поиске дерево не грузим
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasMore := false
	if len(items) > q.Limit {
		hasMore = true
		items = items[:q.Limit]
	}

	return items, hasMore, nil
}
