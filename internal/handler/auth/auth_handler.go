package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"

	"github.com/mihb123/quanly-phongtro/internal/security"
	authsvc "github.com/mihb123/quanly-phongtro/internal/service/auth"
)

type AuthHandler struct {
	service           authsvc.AuthService
	cookieSecure      bool
	trustedProxyCIDRs []netip.Prefix
}

type refreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type registerRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Password  string   `json:"password" validate:"required,min=6"`
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

type AuthHandlerOption func(*AuthHandler)

// WithSecureCookies configures whether auth cookies use the Secure flag.
func WithSecureCookies(secure bool) AuthHandlerOption {
	return func(h *AuthHandler) {
		h.cookieSecure = secure
	}
}

// WithTrustedProxies configures proxy CIDRs trusted for X-Forwarded-* headers
// (client IP extraction and DPoP htu origin).
func WithTrustedProxies(prefixes []netip.Prefix) AuthHandlerOption {
	return func(h *AuthHandler) {
		h.trustedProxyCIDRs = prefixes
	}
}

// NewAuthHandler wires the auth service and security-related HTTP options.
func NewAuthHandler(service authsvc.AuthService, options ...AuthHandlerOption) *AuthHandler {
	h := &AuthHandler{service: service}
	for _, option := range options {
		option(h)
	}
	return h
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := httpx.ValidateStruct(req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
		return
	}

	ipAddress := h.getClientIP(r)
	userAgent := r.UserAgent()
	dpopProof := r.Header.Get("DPoP")

	jkt := ""
	if dpopProof != "" {
		var err error
		jkt, err = security.VerifyDPoPProof(dpopProof, r.Method, security.BuildDPoPHTU(r, h.trustedProxyCIDRs), "")
		if err != nil {
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
			return
		}
	}

	output, err := h.service.Register(r.Context(), authsvc.RegisterInput{
		Email:     strings.TrimSpace(strings.ToLower(req.Email)),
		Password:  strings.TrimSpace(req.Password),
		FullName:  strings.TrimSpace(req.FullName),
		Phone:     strings.TrimSpace(req.Phone),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, sharedsvc.ErrInvalidInput):
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
		case errors.Is(err, authsvc.ErrEmailAlreadyExists):
			httpx.FailWith(w, r, http.StatusConflict, "email already exists", err)
		default:
			httpx.Fail(w, r, err)
		}

		return
	}

	h.setTokenCookies(w, output.AccessToken, output.RefreshToken)
	httpx.WriteJSON(w, http.StatusCreated, output, "registered successfully")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	if err := httpx.ValidateStruct(req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
		return
	}

	ipAddress := h.getClientIP(r)
	userAgent := r.UserAgent()
	dpopProof := r.Header.Get("DPoP")

	jkt := ""
	if dpopProof != "" {
		var err error
		jkt, err = security.VerifyDPoPProof(dpopProof, r.Method, security.BuildDPoPHTU(r, h.trustedProxyCIDRs), "")
		if err != nil {
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
			return
		}
	}

	output, err := h.service.Login(r.Context(), authsvc.LoginInput{
		Email:     strings.TrimSpace(strings.ToLower(req.Email)),
		Password:  strings.TrimSpace(req.Password),
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
	}, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, sharedsvc.ErrInvalidInput):
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
		case errors.Is(err, authsvc.ErrInvalidCredentials):
			httpx.FailWith(w, r, http.StatusUnauthorized, "invalid credentials", err)
		default:
			httpx.Fail(w, r, err)
		}

		return
	}

	h.setTokenCookies(w, output.AccessToken, output.RefreshToken)
	httpx.WriteJSON(w, http.StatusOK, output, "login success")
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		httpx.FailWith(w, r, http.StatusBadRequest, "missing refresh token cookie", errors.New("missing refresh token cookie"))
		return
	}
	refreshTokenStr := cookie.Value

	ipAddress := h.getClientIP(r)
	userAgent := r.UserAgent()
	dpopProof := r.Header.Get("DPoP")

	jkt := ""
	if dpopProof != "" {
		var err error
		jkt, err = security.VerifyDPoPProof(dpopProof, r.Method, security.BuildDPoPHTU(r, h.trustedProxyCIDRs), "")
		if err != nil {
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
			return
		}
	}

	output, err := h.service.RefreshToken(r.Context(), refreshTokenStr, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, sharedsvc.ErrInvalidInput):
			httpx.FailWith(w, r, http.StatusBadRequest, apperr.Message(err), err)
		case errors.Is(err, authsvc.ErrInvalidCredentials):
			httpx.FailWith(w, r, http.StatusUnauthorized, "invalid credentials", err)
		default:
			httpx.Fail(w, r, err)
		}
		return
	}

	h.setTokenCookies(w, output.AccessToken, output.RefreshToken)
	httpx.WriteJSON(w, http.StatusOK, output, "")
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing authentication token")
		return
	}

	userID, err := claims.GetSubject()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "invalid token")
		return
	}

	user, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "user not found")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user, "get user successfully")
}

func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.WriteError(w, http.StatusUnauthorized, "missing authentication token")
		return
	}

	userID, err := claims.GetSubject()
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "invalid token")
		return
	}

	var req authsvc.UpdateProfileInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, "invalid request body", err)
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, user, "profile updated successfully")
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	h.clearTokenCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// clearTokenCookies expires auth cookies using the configured cookie flags.
func (h *AuthHandler) clearTokenCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// setTokenCookies writes auth cookies using the configured cookie flags.
func (h *AuthHandler) setTokenCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})
}

func (h *AuthHandler) CreateOTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.FailWith(w, r, http.StatusInternalServerError, httpx.InternalErrorMessage, errors.New("missing or invalid claims in context"))
		return
	}

	if claims.IsActivated {
		httpx.FailWith(w, r, http.StatusBadRequest, "account is already activated", errors.New("create OTP called on already-activated account"))
		return
	}

	err := h.service.CreateOTP(r.Context(), claims.Email)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, nil, "success")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {

	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		httpx.FailWith(w, r, http.StatusInternalServerError, httpx.InternalErrorMessage, errors.New("missing or invalid claims in context"))
		return
	}

	if claims.IsActivated {
		httpx.FailWith(w, r, http.StatusBadRequest, "account is already activated", errors.New("verify email called on already-activated account"))
		return
	}
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, "invalid body request", err)
		return
	}
	if err := httpx.ValidateStruct(req); err != nil {
		httpx.FailWith(w, r, http.StatusBadRequest, "invalid body request", err)
		return
	}
	isBlock, err := h.service.IsBlockOTP(r.Context(), claims.Email)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	if isBlock {
		httpx.FailWith(w, r, http.StatusBadRequest, "too many request", err)
		return
	}
	jkt := ""
	if claims.Cnf != nil {
		jkt = claims.Cnf["jkt"]
	}

	accessToken, ok, err := h.service.VerifyEmail(r.Context(), claims.Email, req.OTP, jkt)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	if !ok {
		if err := h.service.IncrementOTPCheck(r.Context(), claims.Email); err != nil {
			logging.RecordWarning(r.Context(), "increment OTP check failed", err)
		}
		httpx.FailWith(w, r, http.StatusUnauthorized, "otp is invalid", errors.New("invalid OTP attempt"))
		return
	}

	// Read existing refresh token cookie so we don't clear it
	refreshToken := ""
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	h.setTokenCookies(w, accessToken, refreshToken)
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"access_token": accessToken}, "email verified successfully")
}

// getClientIP returns the client IP, trusting X-Forwarded-For only from configured proxies.
func (h *AuthHandler) getClientIP(r *http.Request) string {
	return security.ResolvedClientIP(r, h.trustedProxyCIDRs)
}
