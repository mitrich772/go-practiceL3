package handlers

import "errors"

// Ошибки, возникающие при валидации запросов хендлеров.
var (
	errInvalidID        = errors.New("invalid id")
	errInvalidLimit     = errors.New("invalid limit (1..500)")
	errInvalidOffset    = errors.New("invalid offset (>=0)")
	errInvalidDate      = errors.New("invalid date format: expected RFC3339 or YYYY-MM-DD")
	errInvalidType      = errors.New("invalid type: expected 'income' or 'expense'")
	errInvalidAmount    = errors.New("amount must be >= 0")
	errCategoryRequired = errors.New("category is required")
	errCategoryTooLong  = errors.New("category must be <= 64 chars")
	errInvalidSort      = errors.New("invalid sort (occurred_at|amount|id)")
	errInvalidOrder     = errors.New("invalid order (asc|desc)")
)
