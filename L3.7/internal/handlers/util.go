package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"warehousecontrol/internal/model"
)

const (
	contentTypeJSON = "application/json; charset=utf-8"
	defaultLimit    = 50
	maxLimit        = 500
)

type ctxKey string

const authUserKey ctxKey = "auth_user"

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

func requestUser(r *http.Request) (model.AuthUser, bool) {
	user, ok := r.Context().Value(authUserKey).(model.AuthUser)
	return user, ok
}

func withUser(ctx context.Context, user model.AuthUser) context.Context {
	return context.WithValue(ctx, authUserKey, user)
}

func requireUser(w http.ResponseWriter, r *http.Request) (model.AuthUser, bool) {
	user, ok := requestUser(r)
	if !ok {
		http.Error(w, errUnauthorized.Error(), http.StatusUnauthorized)
		return model.AuthUser{}, false
	}
	return user, true
}
