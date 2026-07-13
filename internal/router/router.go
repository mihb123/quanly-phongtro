package router

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mihb123/quanly-phongtro/internal/handler/auth"
	"github.com/mihb123/quanly-phongtro/internal/handler/house"
	"github.com/mihb123/quanly-phongtro/internal/handler/httpx"
	"github.com/mihb123/quanly-phongtro/internal/handler/invoice"
	"github.com/mihb123/quanly-phongtro/internal/handler/payment"
	"github.com/mihb123/quanly-phongtro/internal/handler/room"
	"github.com/mihb123/quanly-phongtro/internal/handler/tenant"
	"github.com/mihb123/quanly-phongtro/internal/handler/zalo"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

func New(
	authHandler *auth.AuthHandler,
	houseHandler *house.HouseHandler,
	roomHandler *room.RoomHandler,
	tokenProvider *security.JWTProvider,
	tenantHandler *tenant.TenantHandler,
	invoiceHandler *invoice.InvoiceHandler,
	zaloHandler *zalo.ZaloHandler,
	houseCostHandler *house.HouseCostHandler,
	paymentHandler *payment.PaymentHandler,
	options ...Option,
) *chi.Mux {
	cfg := newOptions(options...)
	r := chi.NewRouter()
	r.Use(recoverMiddleware)
	r.Use(middleware.Logger)

	r.Get("/health", healthCheck)

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/token", authHandler.RefreshToken)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
			r.With(RateLimiter).Get("/verify-email", authHandler.CreateOTP)
			r.Post("/verify-email/otp", authHandler.VerifyEmail)
		})
		r.With(authMiddleware(tokenProvider, cfg.trustedProxies)).Get("/me", authHandler.GetMe)
		r.With(authMiddleware(tokenProvider, cfg.trustedProxies)).Patch("/me", authHandler.UpdateMe)
		r.Post("/logout", authHandler.Logout)
	})

	r.Route("/api/v1/house", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Post("/create", houseHandler.CreateHouse)
		r.Get("/check-code", houseHandler.CheckHouseCode)
		r.Get("/{id}", houseHandler.GetHouseByID)
		r.Get("/", houseHandler.ListHouseByManagerID)
		r.Post("/{id}", houseHandler.UpdateHouse)
		r.Delete("/{id}", houseHandler.DeleteHouse)
	})
	r.Route("/api/v1/room", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Post("/", roomHandler.CreateRoom)
		r.Get("/", roomHandler.ListRooms)
		r.Get("/{id}", roomHandler.GetRoom)
		r.Patch("/{id}", roomHandler.UpdateRoom)
		r.Delete("/{id}", roomHandler.DeleteRoom)
	})

	r.Route("/api/v1/tenant", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Post("/", tenantHandler.RegisterTenant)
		r.Get("/room/{id}", tenantHandler.ListTenantByRoomID)
		r.Get("/house/{id}", tenantHandler.ListTenantByHouseID)
		r.Patch("/{id}", tenantHandler.UpdateTenantInfo)
		r.Delete("/{id}", tenantHandler.DeleteTenant)
		r.Get("/files/*", tenantHandler.DownloadTenantFile)
	})

	r.Route("/api/v1/invoice", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Post("/", invoiceHandler.CreateInvoice)
		r.Get("/", invoiceHandler.ListInvoices)
		r.Get("/{id}", invoiceHandler.GetInvoice)
		r.Get("/{id}/image", invoiceHandler.DownloadInvoiceImage)
		r.Patch("/{id}/pay", invoiceHandler.PayInvoice)
		r.Patch("/{id}/unpay", invoiceHandler.UnpayInvoice)
		r.Delete("/{id}", invoiceHandler.DeleteInvoice)
	})

	r.Route("/api/v1/zalo", func(r chi.Router) {
		r.Post("/webhooks/{managerID}", zaloHandler.Webhook)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
			r.Use(requireRole("MANAGER"))
			r.Get("/public-key", zaloHandler.GetPublicKey)
			r.Get("/config", zaloHandler.GetConfigStatus)
			r.Post("/config", zaloHandler.SaveConfig)
			r.Post("/send-message", zaloHandler.SendMessage)
			r.Post("/invoices/{id}/send", zaloHandler.SendInvoice)
		})
	})

	if paymentHandler != nil {
		r.Route("/api/v1/payments", func(r chi.Router) {
			r.Get("/providers/{provider}/return", paymentHandler.HandleProviderReturn)
			r.Get("/providers/{provider}/cancel", paymentHandler.HandleProviderCancel)
			r.Post("/providers/{provider}/managers/{managerID}/webhook", paymentHandler.HandleProviderWebhook)
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
				r.Use(requireRole("MANAGER"))
				r.Get("/public-key", paymentHandler.GetPublicKey)
				r.Get("/providers/payos/config", paymentHandler.GetPayOSConfig)
				r.Post("/providers/payos/config", paymentHandler.SavePayOSConfig)
				r.Delete("/providers/payos/config", paymentHandler.DeletePayOSConfig)
				r.Get("/providers/sepay/config", paymentHandler.GetSePayConfig)
				r.Post("/providers/sepay/config", paymentHandler.SaveSePayConfig)
				r.Delete("/providers/sepay/config", paymentHandler.DeleteSePayConfig)
				r.Post("/providers/sepay/reconcile", paymentHandler.ReconcileSePay)
			})
		})
		r.Route("/api/v1/payos", func(r chi.Router) {
			r.Get("/return", paymentHandler.HandlePayOSReturn)
			r.Get("/cancel", paymentHandler.HandlePayOSCancel)
			r.Post("/webhook", paymentHandler.HandleLegacyPayOSWebhook)
		})
	}

	r.Route("/api/v1/house-cost", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Post("/", houseCostHandler.CreateMonthlyCost)
		r.Get("/", houseCostHandler.GetMonthlyCost)
		r.Patch("/{id}", houseCostHandler.UpdateMonthlyCost)
	})

	r.Route("/api/v1/revenue-summary", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Get("/", houseCostHandler.GetRevenueSummaries)
	})

	r.Route("/api/v1/uploads/transactions", func(r chi.Router) {
		r.Use(authMiddleware(tokenProvider, cfg.trustedProxies))
		r.Use(requireRole("MANAGER"))
		r.Get("/*", invoiceHandler.DownloadTransactionImage)
	})
	r.Get("/api/v1/uploads/zalo-invoices/*", signedUploadFileHandler("uploads/zalo-invoices", cfg.uploadURLSigningKey))

	// Phục vụ frontend SPA đã nhúng cho mọi request không khớp route API ở trên.
	if cfg.staticFS != nil {
		r.NotFound(spaFileServer(cfg.staticFS))
	}

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
	resdata := httpx.ResData{
		Status:  status,
		Data:    nil,
		Message: message,
	}
	_ = json.NewEncoder(w).Encode(resdata)
}
