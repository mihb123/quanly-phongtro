package auth

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"

	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"

	"github.com/mihb123/quanly-phongtro/internal/security"
	authsvc "github.com/mihb123/quanly-phongtro/internal/service/auth"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
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
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := httpx.ValidateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
			logger.Warn(r, http.StatusBadRequest, "invalid DPoP proof", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
			logger.Warn(r, http.StatusBadRequest, "invalid register input", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, authsvc.ErrEmailAlreadyExists):
			logger.Warn(r, http.StatusConflict, "email already exists", err)
			httpx.WriteError(w, http.StatusConflict, "email already exists")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error during register", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	h.setTokenCookies(w, output.AccessToken, output.RefreshToken)
	httpx.WriteJSON(w, http.StatusCreated, output, "registered successfully")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := httpx.ValidateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
			logger.Warn(r, http.StatusBadRequest, "invalid DPoP proof", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
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
			logger.Warn(r, http.StatusBadRequest, "invalid login input", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, authsvc.ErrInvalidCredentials):
			logger.Warn(r, http.StatusUnauthorized, "invalid credentials", err)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error during login", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
		}

		return
	}

	h.setTokenCookies(w, output.AccessToken, output.RefreshToken)
	httpx.WriteJSON(w, http.StatusOK, output, "login success")
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		logger.Warn(r, http.StatusBadRequest, "missing refresh token cookie", nil)
		httpx.WriteError(w, http.StatusBadRequest, "missing refresh token cookie")
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
			logger.Warn(r, http.StatusBadRequest, "invalid DPoP proof", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	output, err := h.service.RefreshToken(r.Context(), refreshTokenStr, ipAddress, userAgent, jkt)
	if err != nil {
		switch {
		case errors.Is(err, sharedsvc.ErrInvalidInput):
			logger.Warn(r, http.StatusBadRequest, "invalid refresh token input", err)
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, authsvc.ErrInvalidCredentials):
			logger.Warn(r, http.StatusUnauthorized, "invalid refresh token", err)
			httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		default:
			logger.Error(r, http.StatusInternalServerError, "unexpected error during token refresh", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
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
		logger.Warn(r, http.StatusBadRequest, "invalid request body", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "update profile failed", err)
		httpx.WriteError(w, http.StatusInternalServerError, "update profile failed")
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
		logger.Error(r, http.StatusInternalServerError, "missing or invalid claims in context", nil)
		httpx.WriteError(w, http.StatusInternalServerError, "invalid access token")
		return
	}

	if claims.IsActivated {
		logger.Warn(r, http.StatusBadRequest, "create OTP called on already-activated account", nil)
		httpx.WriteError(w, http.StatusBadRequest, "account is already activated")
		return
	}

	err := h.service.CreateOTP(r.Context(), claims.Email)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to create OTP", err)
		httpx.WriteError(w, http.StatusInternalServerError, "cannot create otp")
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, nil, "success")
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {

	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Error(r, http.StatusInternalServerError, "missing or invalid claims in context", nil)
		httpx.WriteError(w, http.StatusInternalServerError, "invalid access token")
		return
	}

	if claims.IsActivated {
		logger.Warn(r, http.StatusBadRequest, "verify email called on already-activated account", nil)
		httpx.WriteError(w, http.StatusBadRequest, "account is already activated")
		return
	}
	var req verifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "invalid body request", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid body request")
		return
	}
	if err := httpx.ValidateStruct(req); err != nil {
		logger.Warn(r, http.StatusBadRequest, "request validation failed", err)
		httpx.WriteError(w, http.StatusBadRequest, "invalid body request")
		return
	}
	isBlock, err := h.service.IsBlockOTP(r.Context(), claims.Email)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to check otp time", err)
		httpx.WriteError(w, http.StatusInternalServerError, "failed to check otp time")
		return
	}

	if isBlock {
		logger.Warn(r, http.StatusBadRequest, "too many request", err)
		httpx.WriteError(w, http.StatusBadRequest, "too many request")
		return
	}
	jkt := ""
	if claims.Cnf != nil {
		jkt = claims.Cnf["jkt"]
	}

	accessToken, ok, err := h.service.VerifyEmail(r.Context(), claims.Email, req.OTP, jkt)
	if err != nil {
		logger.Error(r, http.StatusInternalServerError, "failed to verify email", err)
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !ok {
		logger.Warn(r, http.StatusUnauthorized, "invalid OTP attempt", nil)
		if err := h.service.IncrementOTPCheck(r.Context(), claims.Email); err != nil {
			logger.Error(r, http.StatusInternalServerError, "failed to increment OTP check", err)
		}
		httpx.WriteJSON(w, http.StatusUnauthorized, nil, "otp is invalid")
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
	remoteIP := remoteAddrIP(r.RemoteAddr)
	if remoteIP != "" && h.isTrustedProxy(remoteIP) {
		if xff := firstForwardedIP(r.Header.Get("X-Forwarded-For")); xff != "" {
			return xff
		}
	}

	if remoteIP != "" {
		return remoteIP
	}
	return r.RemoteAddr
}

// isTrustedProxy reports whether an address belongs to a trusted proxy CIDR.
func (h *AuthHandler) isTrustedProxy(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	for _, prefix := range h.trustedProxyCIDRs {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// remoteAddrIP extracts and validates the IP portion of a request RemoteAddr.
func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		if _, parseErr := netip.ParseAddr(remoteAddr); parseErr == nil {
			return remoteAddr
		}
		return ""
	}

	if _, err := netip.ParseAddr(host); err != nil {
		return ""
	}

	return host
}

// firstForwardedIP extracts the first syntactically valid X-Forwarded-For IP.
func firstForwardedIP(header string) string {
	first := strings.TrimSpace(strings.Split(header, ",")[0])
	if first == "" {
		return ""
	}
	if _, err := netip.ParseAddr(first); err != nil {
		return ""
	}
	return first
}
