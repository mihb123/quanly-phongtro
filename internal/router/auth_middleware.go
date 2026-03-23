package router

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/logger"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func authMiddleware(tokens *security.JWTProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				writeUnauthorized(r, w, "missing Authorization header", nil)
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				writeUnauthorized(r, w, "invalid Authorization header format", nil)
				return
			}

			token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
			if token == "" {
				writeUnauthorized(r, w, "missing bearer token", nil)
				return
			}

			claims, err := tokens.Parse(token, "access")
			if err != nil {
				writeUnauthorized(r, w, "invalid token", err)
				return
			}

			ctx := security.WithClaims(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeUnauthorized(r *http.Request, w http.ResponseWriter, message string, err error) {
	logger.Error(r, http.StatusUnauthorized, message, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
