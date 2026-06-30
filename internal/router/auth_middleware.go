package router

import (
	"net/http"
	"net/netip"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
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
				logger.Warn(r, http.StatusBadRequest, "missing authentication token", nil)
				writeError(w, http.StatusBadRequest, "missing authentication token")
				return
			}

			claims, err := tokens.Parse(token, "access")
			if err != nil {
				logger.Warn(r, http.StatusBadRequest, "invalid access token", err)
				writeError(w, http.StatusBadRequest, "invalid token")
				return
			}

			if jkt, ok := claims.Cnf["jkt"]; ok && jkt != "" {
				dpopProof := r.Header.Get("DPoP")
				if dpopProof == "" {
					logger.Warn(r, http.StatusUnauthorized, "missing DPoP proof for DPoP bound token", nil)
					writeError(w, http.StatusUnauthorized, "missing DPoP proof")
					return
				}

				derivedJkt, err := security.VerifyDPoPProof(dpopProof, r.Method, security.BuildDPoPHTU(r, trustedProxies), token)
				if err != nil {
					logger.Warn(r, http.StatusUnauthorized, "invalid DPoP proof", err)
					writeError(w, http.StatusUnauthorized, err.Error())
					return
				}
				if derivedJkt != jkt {
					logger.Warn(r, http.StatusUnauthorized, "invalid DPoP proof: jkt mismatch", nil)
					writeError(w, http.StatusUnauthorized, "invalid DPoP proof: jkt mismatch")
					return
				}
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
				logger.Warn(r, http.StatusUnauthorized, "missing claims in context", nil)
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			for _, role := range roles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			logger.Warn(r, http.StatusForbidden, "insufficient role", nil)
			writeError(w, http.StatusForbidden, "forbidden")
		})
	}
}
