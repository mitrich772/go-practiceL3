package errors

import "errors"

// ErrNotFound возвращается, когда запись не найдена.
var ErrNotFound = errors.New("not found")
