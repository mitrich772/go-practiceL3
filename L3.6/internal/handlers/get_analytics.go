package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"salestracker/internal/model"
)

// AnalyticsGetter возвращает агрегированную аналитику по фильтру.
type AnalyticsGetter interface {
	Analytics(ctx context.Context, f model.ItemFilter) (model.Analytics, error)
}

// GetAnalytics обрабатывает GET /analytics?from=...&to=...
// Согласно ТЗ возвращает: count, sum, avg, median, p90.
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.GetAnalytics"
	log := h.log.With(slog.String("op", op))

	q := r.URL.Query()
	from, err := parseTimeParam(q.Get("from"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to, err := parseTimeParam(q.Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f := model.ItemFilter{From: from, To: to}

	a, err := h.analytics.Analytics(r.Context(), f)
	if err != nil {
		log.Error("failed to compute analytics", slog.Any("err", err))
		http.Error(w, "failed to compute analytics", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, a)
}
