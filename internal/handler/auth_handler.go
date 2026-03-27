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
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "email already exists")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	writeJSON(w, http.StatusCreated, output, "registered successfully")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	output, err := h.service.Login(r.Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusOK, output, "login success")
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	output, err := h.service.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusOK, output, "")
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

func (h *AuthHandler) CreateOTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "invalid access token")
		return
	}

	if claims.IsActivated {
		writeError(w, http.StatusBadRequest, "account is already activated")
		return
	}

	err := h.service.CreateOTP(r.Context(), claims.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot create otp")
		return
	}
	writeJSON(w, http.StatusCreated, nil, "success")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {

	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "invalid access token")
		return
	}

	if claims.IsActivated {
		writeError(w, http.StatusBadRequest, "account is already activated")
		return
	}
	var req struct {
		OTP string `json:"otp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body request")
		return
	}
	if err := validateStruct(req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body request")
		return
	}

	ok, err := h.service.VerifyEmail(r.Context(), claims.Email, req.OTP)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !ok {
		h.service.IncrementOTPCheck(r.Context(), claims.Email)
		writeJSON(w, http.StatusUnauthorized, nil, "otp is invalid")
		return
	}

	writeJSON(w, http.StatusOK, nil, "successs")
}
