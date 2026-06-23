package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"url-shortener/internal/model"
	"url-shortener/internal/service"
)

type URLHandler struct {
	svc *service.URLService
}

func NewURLHandler(svc *service.URLService) *URLHandler {
	return &URLHandler{svc: svc}
}

// POST /api/v1/shorten
func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req model.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp, err := h.svc.Shorten(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAliasExists):
			writeError(w, http.StatusConflict, err.Error())
		default:
			log.Error().Err(err).Msg("Failed to shorten URL")
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"data":    resp,
	})
}

// GET /{shortCode}
func (h *URLHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "shortCode")

	u, err := h.svc.Resolve(r.Context(), code)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, "Short URL not found")
		case errors.Is(err, service.ErrExpired):
			writeError(w, http.StatusGone, "This short URL has expired")
		default:
			writeError(w, http.StatusInternalServerError, "Something went wrong")
		}
		return
	}

	// 301 = permanent (cached by browser), 302 = temporary (not cached)
	// Use 302 so analytics always fires
	http.Redirect(w, r, u.OriginalURL, http.StatusFound)
}

// GET /api/v1/analytics/{shortCode}
func (h *URLHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "shortCode")

	resp, err := h.svc.GetAnalytics(r.Context(), code)
	if err != nil {
		writeError(w, http.StatusNotFound, "Short URL not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data":    resp,
	})
}

// GET /api/v1/health
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "url-shortener",
	})
}

// ── Helpers ───────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("Failed to write JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"success": false,
		"error":   message,
	})
}