package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/store"
)

type recoveryHandlers struct {
	store *store.Store
}

// overview returns everything the Recovery screen needs in one call.
func (h *recoveryHandlers) overview(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())

	profile, err := h.store.GetRecoveryProfile(r.Context(), userID)
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"profile": nil})
		return
	}
	if err != nil {
		slog.Error("recovery profile", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load recovery profile")
		return
	}

	stats, err := h.store.RecoveryStats(r.Context(), userID, profile.SoberSince)
	if err != nil {
		slog.Error("recovery stats", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load recovery stats")
		return
	}
	logs, err := h.store.ListCravingLogs(r.Context(), userID, 50)
	if err != nil {
		slog.Error("craving logs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load craving logs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"profile": profile,
		"stats":   stats,
		"logs":    logs,
	})
}

func (h *recoveryHandlers) upsertProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Focus      string `json:"focus"`
		SoberSince string `json:"soberSince"` // YYYY-MM-DD
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Focus = strings.TrimSpace(req.Focus)
	if req.Focus == "" || len(req.Focus) > 100 {
		writeError(w, http.StatusBadRequest, "focus is required (max 100 chars)")
		return
	}
	soberSince, err := time.Parse("2006-01-02", req.SoberSince)
	if err != nil || soberSince.After(time.Now()) {
		writeError(w, http.StatusBadRequest, "soberSince must be a past date (YYYY-MM-DD)")
		return
	}

	if err := h.store.UpsertRecoveryProfile(r.Context(), auth.UserID(r.Context()), req.Focus, soberSince); err != nil {
		slog.Error("upsert recovery profile", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to save recovery profile")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *recoveryHandlers) deleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteRecoveryProfile(r.Context(), auth.UserID(r.Context())); err != nil {
		slog.Error("delete recovery profile", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete recovery profile")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *recoveryHandlers) logCraving(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Intensity int    `json:"intensity"`
		Trigger   string `json:"trigger"`
		Note      string `json:"note"`
		Lapsed    bool   `json:"lapsed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Intensity < 1 || req.Intensity > 10 {
		writeError(w, http.StatusBadRequest, "intensity must be 1-10")
		return
	}
	req.Trigger = strings.TrimSpace(req.Trigger)
	req.Note = strings.TrimSpace(req.Note)
	if len(req.Trigger) > 100 || len(req.Note) > 1000 {
		writeError(w, http.StatusBadRequest, "trigger or note too long")
		return
	}

	log, err := h.store.CreateCravingLog(r.Context(), auth.UserID(r.Context()), req.Intensity, req.Trigger, req.Note, req.Lapsed)
	if err != nil {
		slog.Error("log craving", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to log craving")
		return
	}
	writeJSON(w, http.StatusCreated, log)
}
