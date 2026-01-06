// Package service содержит бизнес-логику shortener-сервиса.
package service

import "errors"

var (
	// ErrAliasAlreadyExists возвращается, если пользовательский alias уже занят.
	ErrAliasAlreadyExists = errors.New("alias already exists")

	// ErrFailedToGenerateAlias возвращается, если не удалось сгенерировать уникальный alias за допустимое число попыток.
	ErrFailedToGenerateAlias = errors.New("failed to generate alias")
)

// ErrShortNotFound возвращается, если short url отсутствует в хранилище.
var ErrShortNotFound = errors.New("short url not found")
