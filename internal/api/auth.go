package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DiableroTech/fern-server/internal/auth"
	"github.com/DiableroTech/fern-server/internal/store"
)

type authHandlers struct {
	store     *store.Store
	jwtSecret string
}

type userDTO struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"displayName"`
	DefaultModality string `json:"defaultModality"`
}

type authResponse struct {
	User         userDTO `json:"user"`
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
}

func (h *authHandlers) issueTokens(w http.ResponseWriter, r *http.Request, u *store.User) {
	access, err := auth.NewAccessToken(h.jwtSecret, u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	refreshRaw, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	if err := h.store.CreateRefreshToken(r.Context(), u.ID, refreshHash, timeNow().Add(auth.RefreshTokenTTL)); err != nil {
		slog.Error("store refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{
		User:         userDTO{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, DefaultModality: u.DefaultModality},
		AccessToken:  access,
		RefreshToken: refreshRaw,
	})
}

func (h *authHandlers) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "valid email required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}
	u, err := h.store.CreateUser(r.Context(), req.Email, hash, strings.TrimSpace(req.DisplayName))
	if errors.Is(err, store.ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		slog.Error("create user", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}
	h.issueTokens(w, r, u)
}

func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	u, err := h.store.GetUserByEmail(r.Context(), req.Email)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !auth.VerifyPassword(req.Password, u.PasswordHash)) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if err != nil {
		slog.Error("login lookup", "error", err)
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}
	h.issueTokens(w, r, u)
}

func (h *authHandlers) refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refreshToken required")
		return
	}

	userID, err := h.store.ConsumeRefreshToken(r.Context(), auth.HashRefreshToken(req.RefreshToken))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}
	if err != nil {
		slog.Error("consume refresh token", "error", err)
		writeError(w, http.StatusInternalServerError, "refresh failed")
		return
	}

	u, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "account not found")
		return
	}
	h.issueTokens(w, r, u)
}
