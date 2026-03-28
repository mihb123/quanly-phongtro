package router

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func New(authHandler *handler.AuthHandler, houseHandler *handler.HouseHandler, tokens *security.JWTProvider) http.Handler {
	r := chi.NewRouter()
	r.Use(recoverMiddleware)
	r.Use(middleware.Logger)

	r.Get("/health", healthCheck)

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/token", authHandler.RefreshToken)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware(tokens))
			r.With(RateLimiter).Get("/verify-email", authHandler.CreateOTP)
			r.Post("/verify-email/otp", authHandler.VerifyEmail)
		})
		r.With(authMiddleware(tokens)).Get("/me", authHandler.GetMe)
		r.Post("/logout", authHandler.Logout)
	})

	r.Route("/api/v1/house", func(r chi.Router) {
		r.Use(authMiddleware(tokens))
		r.Post("/create", houseHandler.CreateHouse)
		r.Get("/{id}", houseHandler.GetHouseByID)
	})

	return r
}

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	resdata := handler.ResData{
		Status:  status,
		Data:    nil,
		Message: message,
	}
	_ = json.NewEncoder(w).Encode(resdata)
}
