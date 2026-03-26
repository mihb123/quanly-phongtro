package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
)

type AuthHandler struct {
	service service.AuthService
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type registerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func NewAuthHandler(service service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error(), err)
		return
	}

	output, err := h.service.Register(r.Context(), service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(r, w, http.StatusBadRequest, err.Error(), err)
		case errors.Is(err, service.ErrEmailAlreadyExists):
			writeError(r, w, http.StatusConflict, "email already exists", err)
		default:
			writeError(r, w, http.StatusInternalServerError, "internal server error", err)
		}

		return
	}

	writeJSON(w, http.StatusCreated, output)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error(), err)
		return
	}

	output, err := h.service.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(r, w, http.StatusBadRequest, err.Error(), err)
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(r, w, http.StatusUnauthorized, "invalid credentials", err)
		default:
			writeError(r, w, http.StatusInternalServerError, "internal server error", err)
		}

		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusOK, output)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(r, w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(r, w, http.StatusBadRequest, err.Error(), err)
		return
	}

	output, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(r, w, http.StatusBadRequest, err.Error(), err)
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(r, w, http.StatusUnauthorized, "invalid credentials", err)
		default:
			writeError(r, w, http.StatusInternalServerError, "internal server error", err)
		}
		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusOK, output)
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(r, w, http.StatusUnauthorized, "missing authentication token", nil)
		return
	}

	user, err := h.service.GetMe(r.Context(), claims.UserID)
	if err != nil {
		writeError(r, w, http.StatusUnauthorized, "user not found", err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	clearTokenCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func clearTokenCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})
}
