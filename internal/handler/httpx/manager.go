package httpx

import (
	"errors"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/security"
)

// GetManagerID extracts the authenticated manager's user ID from the request
// claims, writing an unauthorized response and returning false when absent.
func GetManagerID(r *http.Request, w http.ResponseWriter) (string, bool) {
	claims, ok := security.ClaimsFromContext(r.Context())
	if !ok || claims == nil {
		FailWith(w, r, http.StatusUnauthorized, "unauthorized", errors.New("unauthorized: missing or invalid claims"))
		return "", false
	}
	userID, err := claims.GetSubject()
	if err != nil {
		FailWith(w, r, http.StatusUnauthorized, "unauthorized", err)
		return "", false
	}
	return userID, true
}
