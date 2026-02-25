package model

import "time"

type Status string

const (
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusFailed     Status = "failed"
)

// Image — метаданные изображения (то, что лежит в таблице images).
type Image struct {
	ID            string
	Status        Status
	OriginalPath  string
	ProcessedPath *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
