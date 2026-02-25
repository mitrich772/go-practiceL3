package dto

type ImageJob struct {
	ID           string `json:"id"`
	OriginalPath string `json:"original_path"`

	Mode          string `json:"mode"` // resize|thumb|watermark
	Width         int    `json:"width,omitempty"`
	Height        int    `json:"height,omitempty"`
	WatermarkText string `json:"watermark_text,omitempty"`
}
