package router

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// authMiddleware validates the access token and, for DPoP-bound tokens, verifies
// the DPoP proof against the public request origin (scheme/host derived from the
// request, trusting X-Forwarded-* only from the given proxies).
func authMiddleware(tokens *security.JWTProvider, trustedProxies []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// 1. Try to get token from Authorization header
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader != "" {
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
				} else if strings.HasPrefix(authHeader, "DPoP ") {
					token = strings.TrimSpace(strings.TrimPrefix(authHeader, "DPoP "))
				}
			}

			// 2. Try to get token from cookie if header is missing
			if token == "" {
				cookie, err := r.Cookie("access_token")
				if err == nil {
					token = cookie.Value
				}
			}

			if token == "" {
				reject(w, r, http.StatusBadRequest, "missing authentication token", "missing authentication token", nil)
				return
			}

			claims, err := tokens.Parse(token, "access")
			if err != nil {
				reject(w, r, http.StatusBadRequest, "invalid token", "invalid access token", err)
				return
			}

			if jkt, ok := claims.Cnf["jkt"]; ok && jkt != "" {
				dpopProof := r.Header.Get("DPoP")
				if dpopProof == "" {
					reject(w, r, http.StatusUnauthorized, "missing DPoP proof", "missing DPoP proof for DPoP bound token", nil)
					return
				}

				derivedJkt, err := security.VerifyDPoPProof(dpopProof, r.Method, security.BuildDPoPHTU(r, trustedProxies), token)
				if err != nil {
					reject(w, r, http.StatusUnauthorized, err.Error(), "invalid DPoP proof", err)
					return
				}
				if derivedJkt != jkt {
					reject(w, r, http.StatusUnauthorized, "invalid DPoP proof: jkt mismatch", "invalid DPoP proof: jkt mismatch", nil)
					return
				}
			}

			if sub, err := claims.GetSubject(); err == nil {
				logging.SetUserID(r.Context(), sub)
			}
			ctx := security.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := security.ClaimsFromContext(r.Context())
			if !ok || claims == nil {
				reject(w, r, http.StatusUnauthorized, "unauthorized", "missing claims in context", nil)
				return
			}

			for _, role := range roles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			reject(w, r, http.StatusForbidden, "forbidden", "insufficient role", nil)
		})
	}
}

// reject trả lỗi xác thực cho client và ghi lý do vào dòng log của request (WARN).
func reject(w http.ResponseWriter, r *http.Request, status int, clientMsg, reason string, err error) {
	if err != nil {
		err = fmt.Errorf("%s: %w", reason, err)
	} else {
		err = errors.New(reason)
	}
	logging.RecordError(r.Context(), err)
	writeError(w, status, clientMsg)
}
