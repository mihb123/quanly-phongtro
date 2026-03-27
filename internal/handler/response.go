package handler

import (
	"encoding/json"
	"net/http"
)

type ResData struct {
	Status  int    `json:"status"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data any, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resdata := ResData{
		Status:  status,
		Data:    data,
		Message: message,
	}
	_ = json.NewEncoder(w).Encode(resdata)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, nil, message)
}
