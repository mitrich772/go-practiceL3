package dto

type ShortenRequest struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty" validate:"omitempty,alphanum"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}
