package handler

import (
	"encoding/json"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/logger"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(r *http.Request, w http.ResponseWriter, status int, message string, err error) {
	logger.Error(r, status, message, err)
	writeJSON(w, status, map[string]string{"error": message})
}
