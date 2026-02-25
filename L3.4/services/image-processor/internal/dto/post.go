package dto

import "time"

// UploadRequestDTO — что мы извлекаем из multipart/form-data (file + поля формы).
// Важно: это не "авто-декод" как JSON, мы заполняем struct вручную из r.FormFile / r.FormValue.
type UploadRequestDTO struct {
	// File
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Data        []byte `json:"-"` // бинарь в ответ не возвращаем

	// Options (из form полей)
	Mode          string `json:"mode"`           // resize|thumb|watermark (опционально)
	Width         int    `json:"width"`          // опционально
	Height        int    `json:"height"`         // опционально
	WatermarkText string `json:"watermark_text"` // опционально
}

// UploadResponseDTO — ответ API после принятия задачи.
type UploadResponseDTO struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"` // processing
	CreatedAt time.Time `json:"created_at"`

	// (для отладки/фронта удобно вернуть мету файла)
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Mode        string `json:"mode"`
}
