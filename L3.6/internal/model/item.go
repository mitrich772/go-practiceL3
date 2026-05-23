// Package model содержит доменные сущности SalesTracker.
package model

import "time"

// ItemType описывает направление денежного потока: доход или расход.
type ItemType string

const (
	// ItemTypeIncome — поступление (доход).
	ItemTypeIncome ItemType = "income"
	// ItemTypeExpense — списание (расход).
	ItemTypeExpense ItemType = "expense"
)

// Valid проверяет, что значение типа допустимо.
func (t ItemType) Valid() bool {
	return t == ItemTypeIncome || t == ItemTypeExpense
}

// Item — финансовая запись (доход или расход).
type Item struct {
	ID         int64     `json:"id"          db:"id"`
	Type       ItemType  `json:"type"        db:"type"`
	Amount     float64   `json:"amount"      db:"amount"`
	Category   string    `json:"category"    db:"category"`
	Note       string    `json:"note"        db:"note"`
	OccurredAt time.Time `json:"occurred_at" db:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"  db:"created_at"`
}

// ItemFilter описывает параметры выборки записей.
type ItemFilter struct {
	From     *time.Time
	To       *time.Time
	Type     ItemType
	Category string
	Limit    int
	Offset   int
	Sort     string // occurred_at | amount | id
	Order    string // asc | desc
}

// Analytics — агрегированная сводка за период согласно ТЗ:
// count, sum, avg, median, 90-й перцентиль.
type Analytics struct {
	From         *time.Time `json:"from,omitempty"`
	To           *time.Time `json:"to,omitempty"`
	Count        int64      `json:"count"`
	Sum          float64    `json:"sum"`
	Avg          float64    `json:"avg"`
	Median       float64    `json:"median"`
	Percentile90 float64    `json:"p90"`
}
