package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"salestracker/internal/model"
)

type CSVLister interface {
	List(ctx context.Context, f model.ItemFilter) ([]model.Item, bool, error)
}

const csvPageSize = 1000

func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.ExportCSV"
	log := h.log.With(slog.String("op", op))

	f, err := parseItemFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.Limit = csvPageSize
	f.Offset = 0
	f.Sort = "occurred_at"
	f.Order = "asc"

	filename := fmt.Sprintf("items_%s.csv", time.Now().UTC().Format("20060102_150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	_, err = w.Write([]byte{0xEF, 0xBB, 0xBF})
	if err != nil {
		log.Error("failed to write csv BOM", slog.Any("err", err))
		return
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"id", "type", "amount", "category", "note", "occurred_at", "created_at"}
	if err := cw.Write(header); err != nil {
		log.Error("failed to write csv header", slog.Any("err", err))
		return
	}

	offset := 0
	for {
		f.Offset = offset
		items, hasMore, err := h.lister.List(r.Context(), f)
		if err != nil {
			log.Error("failed to list items for csv", slog.Any("err", err))
			return
		}
		for _, it := range items {
			row := []string{
				strconv.FormatInt(it.ID, 10),
				string(it.Type),
				strconv.FormatFloat(it.Amount, 'f', 2, 64),
				it.Category,
				it.Note,
				it.OccurredAt.UTC().Format(time.RFC3339),
				it.CreatedAt.UTC().Format(time.RFC3339),
			}
			if err := cw.Write(row); err != nil {
				log.Error("failed to write csv row", slog.Any("err", err))
				return
			}
		}
		cw.Flush()
		if !hasMore {
			break
		}
		offset += csvPageSize
	}
}
