package httpx

import (
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

// GetManagerID extracts the authenticated manager's user ID from the request
// claims, writing an unauthorized response and returning false when absent.
func GetManagerID(r *http.Request, w http.ResponseWriter) (string, bool) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: missing or invalid claims", nil)
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	userID, err := claims.GetSubject()
	if err != nil {
		logger.Warn(r, http.StatusUnauthorized, "unauthorized: cannot read subject", err)
		WriteError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	return userID, true
}
