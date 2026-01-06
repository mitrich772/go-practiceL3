// Package handlers содержит HTTP-обработчики (эндпоинты) приложения.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"analytics/internal/handlers/dto"
	"analytics/internal/service"

	"github.com/go-chi/chi/v5"
)

// Handler объединяет зависимости HTTP-слоя: логгер и сервис аналитики.
type Handler struct {
	log       *slog.Logger
	analytics *service.AnalyticsService
}

// New создаёт новый Handler.
func New(log *slog.Logger, analytics *service.AnalyticsService) *Handler {
	return &Handler{
		log:       log,
		analytics: analytics,
	}
}

// Click принимает событие клика по короткой ссылке и сохраняет его.
func (h *Handler) Click(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.click"
	log := h.log.With(slog.String("op", op))

	var req dto.ClickEvent
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid json", slog.Any("error", err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Short == "" {
		http.Error(w, "short is required", http.StatusBadRequest)
		return
	}

	if err := h.analytics.SaveClick(
		r.Context(),
		req.Short,
		req.IP,
		req.UserAgent,
		req.Referer,
	); err != nil {
		log.Error("failed to save click", slog.Any("error", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	log.Info(
		"click saved",
		slog.String("short", req.Short),
		slog.String("ip", req.IP),
	)

	w.WriteHeader(http.StatusNoContent)
}

// GetAnalytics возвращает агрегированную статистику кликов по short.
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.getAnalytics"
	log := h.log.With(slog.String("op", op))

	short := chi.URLParam(r, "short")
	if short == "" {
		http.Error(w, "short is required", http.StatusBadRequest)
		return
	}

	total, clicks, err := h.analytics.GetStats(r.Context(), short)
	if err != nil {
		log.Error("failed to get analytics", slog.Any("error", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := dto.AnalyticsResponse{
		Short: short,
		Total: total,
	}

	for _, c := range clicks {
		resp.Clicks = append(resp.Clicks, dto.ClickEntry{
			ClickedAt: c.ClickedAt,
			UserAgent: c.UserAgent,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// Helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
	}
}
