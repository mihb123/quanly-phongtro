package httpx

import (
	"encoding/json"
	"net/http"
)

type ResData struct {
	Status  int    `json:"status"`
	Data    any    `json:"data,omitempty"`
	Message string `json:"message,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, data any, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resdata := ResData{
		Status:  status,
		Data:    data,
		Message: message,
	}
	_ = json.NewEncoder(w).Encode(resdata)
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, nil, message)
}
