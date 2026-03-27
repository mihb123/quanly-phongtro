package router

import (
	"net/http"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/logger"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func authMiddleware(tokens *security.JWTProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// 1. Try to get token from Authorization header
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader != "" {
				const prefix = "Bearer "
				if strings.HasPrefix(authHeader, prefix) {
					token = strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
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

			ctx := security.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
