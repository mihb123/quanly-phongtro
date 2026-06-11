package handler

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
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
	FullName  string   `json:"full_name"`
	Phone     string   `json:"phone"`
	Latitude  *float64 `json:"Latitude"`
	Longitude *float64 `json:"Longitude"`
}

type loginRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Password  string   `json:"password" validate:"required"`
	Latitude  *float64 `json:"Latitude"`
	Longitude *float64 `json:"Longitude"`
}

type verifyEmailRequest struct {
	OTP string `json:"otp" validate:"required"`
}

func NewAuthHandler(service service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()
	dpopProof := r.Header.Get("DPoP")
	
	jkt := ""
	if dpopProof != "" {
		var err error
		jkt, err = security.VerifyDPoPProof(dpopProof, r.Method, r.URL.Path, "")
		if err != nil {
			logger.Warn(r, http.StatusBadRequest, "invalid DPoP proof", err)
			writeError(w, http.StatusBadRequest, "invalid DPoP proof")
			return
		}
	}

	output, err := h.service.Register(r.Context(), service.RegisterInput{
		Email:     strings.TrimSpace(strings.ToLower(req.Email)),
		Password:  strings.TrimSpace(req.Password),
		FullName:  strings.TrimSpace(req.FullName),
		Phone:     strings.TrimSpace(req.Phone),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			logger.Warn(r, http.StatusBadRequest, "invalid register input", err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrEmailAlreadyExists):
			logger.Warn(r, http.StatusConflict, "email already exists", err)
			writeError(w, http.StatusConflict, "email already exists")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error during register", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusCreated, output, "registered successfully")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()
	dpopProof := r.Header.Get("DPoP")
	
	jkt := ""
	if dpopProof != "" {
		var err error
		jkt, err = security.VerifyDPoPProof(dpopProof, r.Method, r.URL.Path, "")
		if err != nil {
			logger.Warn(r, http.StatusBadRequest, "invalid DPoP proof", err)
			writeError(w, http.StatusBadRequest, "invalid DPoP proof")
			return
		}
	}

	output, err := h.service.Login(r.Context(), service.LoginInput{
		Email:     strings.TrimSpace(strings.ToLower(req.Email)),
		Password:  strings.TrimSpace(req.Password),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			logger.Warn(r, http.StatusBadRequest, "invalid login input", err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			logger.Warn(r, http.StatusUnauthorized, "invalid credentials", err)
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error during login", err)
			writeError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusOK, output, "login success")
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		logger.Warn(r, http.StatusBadRequest, "missing refresh token cookie", nil)
		writeError(w, http.StatusBadRequest, "missing refresh token cookie")
		return
	}
	refreshTokenStr := cookie.Value

	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()
	dpopProof := r.Header.Get("DPoP")
	
	jkt := ""
	if dpopProof != "" {
		var err error
		jkt, err = security.VerifyDPoPProof(dpopProof, r.Method, r.URL.Path, "")
		if err != nil {
			logger.Warn(r, http.StatusBadRequest, "invalid DPoP proof", err)
			writeError(w, http.StatusBadRequest, "invalid DPoP proof")
			return
		}
	}

	output, err := h.service.RefreshToken(r.Context(), refreshTokenStr, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			logger.Warn(r, http.StatusBadRequest, "invalid refresh token input", err)
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrInvalidCredentials):
			logger.Warn(r, http.StatusUnauthorized, "invalid refresh token", err)
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error during token refresh", err)
			writeError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	setTokenCookies(w, output.AccessToken, output.RefreshToken)
	writeJSON(w, http.StatusOK, output, "")
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing authentication token")
		return
	}

	userID, err := claims.GetSubject()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "invalid token")
	}

	user, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user, "get user successfully")
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
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

func (h *AuthHandler) CreateOTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		logger.Error(r, http.StatusInternalServerError, "missing or invalid claims in context", nil)
		writeError(w, http.StatusInternalServerError, "invalid access token")
		return
	}

	if claims.IsActivated {
		logger.Warn(r, http.StatusBadRequest, "create OTP called on already-activated account", nil)
		writeError(w, http.StatusBadRequest, "account is already activated")
		return
	}

	err := h.service.CreateOTP(r.Context(), claims.Email)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to create OTP", err)
		writeError(w, http.StatusInternalServerError, "cannot create otp")
		return
	}
	writeJSON(w, http.StatusCreated, nil, "success")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {

	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok {
		logger.Error(r, http.StatusInternalServerError, "missing or invalid claims in context", nil)
		writeError(w, http.StatusInternalServerError, "invalid access token")
		return
	}

	if claims.IsActivated {
		logger.Warn(r, http.StatusBadRequest, "verify email called on already-activated account", nil)
		writeError(w, http.StatusBadRequest, "account is already activated")
		return
	}
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid body request", err)
		writeError(w, http.StatusBadRequest, "invalid body request")
		return
	}
	if err := validateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		writeError(w, http.StatusBadRequest, "invalid body request")
		return
	}
	isBlock, err := h.service.IsBlockOTP(r.Context(), claims.Email)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to check otp time", err)
		writeError(w, http.StatusInternalServerError, "failed to check otp time")
		return
	}

	if isBlock {
		logger.Warn(r, http.StatusBadRequest, "too many request", err)
		writeError(w, http.StatusBadRequest, "too many request")
		return
	}
	jkt := ""
	if claims.Cnf != nil {
		jkt = claims.Cnf["jkt"]
	}

	accessToken, ok, err := h.service.VerifyEmail(r.Context(), claims.Email, req.OTP, jkt)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to verify email", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !ok {
		logger.Warn(r, http.StatusUnauthorized, "invalid OTP attempt", nil)
		if err := h.service.IncrementOTPCheck(r.Context(), claims.Email); err != nil {
			logger.Error(r, http.StatusInternalServerError, "failed to increment OTP check", err)
		}
		writeJSON(w, http.StatusUnauthorized, nil, "otp is invalid")
		return
	}

	// Read existing refresh token cookie so we don't clear it
	refreshToken := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	setTokenCookies(w, accessToken, refreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"access_token": accessToken}, "email verified successfully")
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.Split(xff, ",")[0]
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Fallback to raw if not ip:port
	}
	return host
}
