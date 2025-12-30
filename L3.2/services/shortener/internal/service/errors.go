package service

import "errors"

var (
	ErrAliasAlreadyExists    = errors.New("alias already exists")
	ErrFailedToGenerateAlias = errors.New("failed to generate alias")
)

var ErrShortNotFound = errors.New("short url not found")
