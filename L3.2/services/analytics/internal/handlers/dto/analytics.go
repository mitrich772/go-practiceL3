// Package dto содержит структуры запросов/ответов HTTP API.
package dto

import "time"

// ClickEvent описывает входящее событие клика (request body).
type ClickEvent struct {
	Short     string `json:"short"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
}

// ClickEntry описывает один клик в ответе аналитики.
type ClickEntry struct {
	ClickedAt time.Time `json:"clicked_at"`
	UserAgent string    `json:"user_agent"`
}

// AnalyticsResponse описывает ответ эндпоинта аналитики.
type AnalyticsResponse struct {
	Short  string       `json:"short"`
	Total  int64        `json:"total"`
	Clicks []ClickEntry `json:"clicks"`
}
