package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	contentTypeJSON = "application/json; charset=utf-8"
	defaultLimit    = 50
	maxLimit        = 500
)

// writeJSON отправляет ответ в JSON с заданным статусом.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
	}
}

// parseTimeParam парсит RFC3339 / "2006-01-02" / "2006-01-02 15:04:05" в *time.Time.
// Пустая строка → (nil, nil).
func parseTimeParam(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t, nil
		}
	}
	return nil, errInvalidDate
}

// parseLimitOffset извлекает limit/offset из query параметров.
func parseLimitOffset(r *http.Request) (int, int, error) {
	limit := defaultLimit
	offset := 0

	if s := strings.TrimSpace(r.URL.Query().Get("limit")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v <= 0 || v > maxLimit {
			return 0, 0, errInvalidLimit
		}
		limit = v
	}

	if s := strings.TrimSpace(r.URL.Query().Get("offset")); s != "" {
		v, err := strconv.Atoi(s)
		if err != nil || v < 0 {
			return 0, 0, errInvalidOffset
		}
		offset = v
	}

	return limit, offset, nil
}

// parseIDPath парсит {id} из URL.
func parseIDPath(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errInvalidID
	}
	return id, nil
}
