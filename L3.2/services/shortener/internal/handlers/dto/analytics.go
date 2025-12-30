package dto

type AnalyticsResponseDTO struct {
	Short string `json:"short"`
	Total int64  `json:"total"`

	Period *struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period,omitempty"`
}

type ClickEvent struct {
	Short     string `json:"short"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Referer   string `json:"referer"`
}
