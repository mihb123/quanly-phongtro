package router

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func New(authHandler *handler.AuthHandler, houseHandler *handler.HouseHandler, tokens *security.JWTProvider) *mux.Router {
	r := mux.NewRouter()
	r.Use(recoverMiddleware)

	r.HandleFunc("/health", healthCheck).Methods(http.MethodGet)

	authRoute := r.PathPrefix("/api/v1/auth").Subrouter()
	authRoute.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	authRoute.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	houseRoute := r.PathPrefix("/api/v1/house").Subrouter()
	houseRoute.Use(authMiddleware(tokens))
	houseRoute.HandleFunc("/create", houseHandler.CreateHouse).Methods(http.MethodPost)
	houseRoute.HandleFunc("/{id}", houseHandler.GetHouseByID).Methods(http.MethodGet)

	return r
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}
