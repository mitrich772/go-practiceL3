package dto

import "time"

type ClickEvent struct {
	Short     string `json:"short"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
}

type ClickEntry struct {
	ClickedAt time.Time `json:"clicked_at"`
	UserAgent string    `json:"user_agent"`
}

type AnalyticsResponse struct {
	Short  string       `json:"short"`
	Total  int64        `json:"total"`
	Clicks []ClickEntry `json:"clicks"`
}
