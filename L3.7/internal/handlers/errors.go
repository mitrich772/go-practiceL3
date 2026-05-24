package handlers

import "errors"

// Ошибки, возникающие при валидации запросов хендлеров.
var (
	errInvalidID             = errors.New("invalid id")
	errInvalidLimit          = errors.New("invalid limit (1..500)")
	errInvalidOffset         = errors.New("invalid offset (>=0)")
	errInvalidName           = errors.New("name is required and must be <= 120 chars")
	errInvalidSKU            = errors.New("sku is required and must be <= 64 chars")
	errInvalidQuantity       = errors.New("quantity must be >= 0")
	errInvalidLocation       = errors.New("location must be <= 120 chars")
	errInvalidSort           = errors.New("invalid sort (id|name|sku|quantity|updated_at)")
	errInvalidOrder          = errors.New("invalid order (asc|desc)")
	errUnauthorized          = errors.New("unauthorized")
	errForbidden             = errors.New("forbidden")
	errInvalidRole           = errors.New("invalid role")
	errInvalidToken          = errors.New("invalid token")
	errAuthorizationRequired = errors.New("authorization header is required")
)
