package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/prompts"
	"github.com/DiableroTech/fern-server/internal/store"
)

type meHandlers struct {
	store *store.Store
}

type meDTO struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"displayName"`
	DefaultModality string `json:"defaultModality"`
	CreatedAt       string `json:"createdAt"`
}

func toMeDTO(u *store.User) meDTO {
	return meDTO{
		ID:              u.ID,
		Email:           u.Email,
		DisplayName:     u.DisplayName,
		DefaultModality: u.DefaultModality,
		CreatedAt:       u.CreatedAt.Format("2006-01-02"),
	}
}

func (h *meHandlers) get(w http.ResponseWriter, r *http.Request) {
	u, err := h.store.GetUserByID(r.Context(), auth.UserID(r.Context()))
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, toMeDTO(u))
}

func (h *meHandlers) update(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	u, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}

	var req struct {
		DisplayName     *string `json:"displayName"`
		DefaultModality *string `json:"defaultModality"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	displayName := u.DisplayName
	if req.DisplayName != nil {
		displayName = strings.TrimSpace(*req.DisplayName)
		if len(displayName) > 100 {
			writeError(w, http.StatusBadRequest, "display name too long")
			return
		}
	}
	modality := u.DefaultModality
	if req.DefaultModality != nil {
		if _, err := prompts.System(prompts.Modality(*req.DefaultModality), ""); err != nil {
			writeError(w, http.StatusBadRequest, "unknown modality")
			return
		}
		modality = *req.DefaultModality
	}

	if err := h.store.UpdateUser(r.Context(), userID, displayName, modality); err != nil {
		slog.Error("update user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	u.DisplayName = displayName
	u.DefaultModality = modality
	writeJSON(w, http.StatusOK, toMeDTO(u))
}

// export returns everything Fern knows about this person as one JSON file.
func (h *meHandlers) export(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := auth.UserID(ctx)

	u, err := h.store.GetUserByID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	sessions, err := h.store.ExportSessions(ctx, userID)
	if err != nil {
		slog.Error("export sessions", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	logs, err := h.store.ListCravingLogs(ctx, userID, 100000)
	if err != nil {
		slog.Error("export cravings", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	reports, err := h.store.ListReports(ctx, userID, 100000)
	if err != nil {
		slog.Error("export reports", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}
	var recovery *store.RecoveryProfile
	if p, err := h.store.GetRecoveryProfile(ctx, userID); err == nil {
		recovery = p
	} else if !errors.Is(err, store.ErrNotFound) {
		slog.Error("export recovery", "error", err)
		writeError(w, http.StatusInternalServerError, "export failed")
		return
	}

	w.Header().Set("Content-Disposition", `attachment; filename="fern-export.json"`)
	writeJSON(w, http.StatusOK, map[string]any{
		"exportedAt":      timeNow().Format("2006-01-02T15:04:05Z07:00"),
		"account":         toMeDTO(u),
		"sessions":        sessions,
		"recoveryProfile": recovery,
		"cravingLogs":     logs,
		"reports":         reports,
	})
}

func (h *meHandlers) delete(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserID(r.Context())
	if err := h.store.DeleteUser(r.Context(), userID); err != nil {
		slog.Error("delete user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete account")
		return
	}
	slog.Info("account deleted", "user", userID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
