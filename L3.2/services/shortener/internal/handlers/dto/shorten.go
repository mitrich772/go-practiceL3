// Package dto содержит структуры запросов/ответов HTTP API.
package dto

// ShortenRequest описывает запрос на создание короткой ссылки.
type ShortenRequest struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty" validate:"omitempty,alphanum"`
}

// ShortenResponse описывает ответ на создание короткой ссылки.
type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}
