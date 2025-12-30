package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"shortener/internal/handlers/dto"
	"shortener/internal/handlers/validation"
	"shortener/internal/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	log *slog.Logger

	analyticsURL string
	httpClient   *http.Client

	shortener  *service.ShortenerService
	redirector *service.RedirectService
}

func New(
	log *slog.Logger,
	shortener *service.ShortenerService,
	redirector *service.RedirectService,
	analyticsURL string,
) *Handler {
	return &Handler{
		log:          log,
		shortener:    shortener,
		redirector:   redirector,
		analyticsURL: analyticsURL,
		httpClient: &http.Client{
			Timeout: 500 * time.Millisecond,
		},
	}
}

// Handlers
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.shorten"
	log := h.log.With(
		slog.String("op", op),
	)

	var req dto.ShortenRequest

	// json parse to dto.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Warn("invalid json",
			slog.Any("error", err),
		)
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	// validate
	if err := validation.ValidateShorten(req); err != nil {
		log.Warn("validation error",
			slog.Any("error", err),
		)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// save
	alias, err := h.shortener.Shorten(
		r.Context(),
		req.URL,
		req.Alias,
	)
	// check save error
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAliasAlreadyExists):
			log.Warn("alias already exists",
				slog.Any("error", err),
			)
			http.Error(w, "alias already exists", http.StatusConflict)

		case errors.Is(err, service.ErrFailedToGenerateAlias):
			log.Error("alias generation exhausted",
				slog.Any("error", err),
			)
			http.Error(w, "failed to generate short url", http.StatusInternalServerError)

		default:
			log.Error("shorten failed",
				slog.Any("error", err),
			)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// make response
	resp := dto.ShortenResponse{
		ShortURL: getBaseURL(r) + "/s/" + alias,
	}

	log.Info("url shortened",
		slog.String("short", alias),
	)

	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.redirect"
	log := h.log.With(slog.String("op", op))

	short := chi.URLParam(r, "short_url")
	originalURL, err := h.redirector.Resolve(r.Context(), short)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrShortNotFound):
			log.Warn("short url not found",
				slog.String("short", short),
			)
			http.Error(w, "not found", http.StatusNotFound)

		default:
			log.Error("redirect failed",
				slog.Any("error", err),
			)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	// stat send
	go h.sendClickEvent(r, short)

	log.Info("redirect",
		slog.String("short", short),
	)

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func (h *Handler) ProxyAnalytics(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.proxyAnalytics"
	log := h.log.With(slog.String("op", op))

	short := chi.URLParam(r, "short_url")
	if short == "" {
		log.Warn("short url is empty")
		http.Error(w, "short is required", http.StatusBadRequest)
		return
	}

	url := h.analyticsURL + "/analytics/" + short
	if r.URL.RawQuery != "" {
		url += "?" + r.URL.RawQuery
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Warn(
			"failed to create analytics request",
			slog.Any("error", err),
		)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Warn(
			"analytics unavailable",
			slog.Any("error", err),
		)
		http.Error(w, "analytics unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Warn(
			"failed to copy analytics response",
			slog.Any("error", err),
		)
		return
	}

	log.Info(
		"analytics request proxied",
		slog.String("short", short),
		slog.Int("status", resp.StatusCode),
	)
}

// Helpers

func (h *Handler) sendClickEvent(r *http.Request, short string) {
	const op = "handlers.sendClickEvent"

	log := h.log.With(
		slog.String("op", op),
		slog.String("short", short),
	)

	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		ip = host
	}

	event := dto.ClickEvent{
		Short:     short,
		IP:        ip,
		UserAgent: r.UserAgent(),
		Referer:   r.Referer(),
	}

	body, err := json.Marshal(event)
	if err != nil {
		log.Warn("failed to marshal click event",
			slog.Any("error", err),
		)
		return
	}

	req, err := http.NewRequest(
		http.MethodPost,
		h.analyticsURL+"/events/click",
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Warn("failed to create analytics request",
			slog.Any("error", err),
		)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Warn("failed to send click event to analytics",
			slog.Any("error", err),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Warn("analytics responded with non-2xx status",
			slog.Int("status", resp.StatusCode),
		)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "json encode error", http.StatusInternalServerError)
	}
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if forwarded := r.Header.Get("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}

	return scheme + "://" + r.Host
}
