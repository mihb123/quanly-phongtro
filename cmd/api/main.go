package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mihb123/quanly-phongtro/config"
	"github.com/mihb123/quanly-phongtro/internal/assets"
	"github.com/mihb123/quanly-phongtro/internal/db"
	authhandler "github.com/mihb123/quanly-phongtro/internal/handler/auth"
	househandler "github.com/mihb123/quanly-phongtro/internal/handler/house"
	invoicehandler "github.com/mihb123/quanly-phongtro/internal/handler/invoice"
	paymenthandler "github.com/mihb123/quanly-phongtro/internal/handler/payment"
	roomhandler "github.com/mihb123/quanly-phongtro/internal/handler/room"
	tenanthandler "github.com/mihb123/quanly-phongtro/internal/handler/tenant"
	zalohandler "github.com/mihb123/quanly-phongtro/internal/handler/zalo"
	authrepo "github.com/mihb123/quanly-phongtro/internal/repository/auth"
	houserepo "github.com/mihb123/quanly-phongtro/internal/repository/house"
	invoicerepo "github.com/mihb123/quanly-phongtro/internal/repository/invoice"
	revenuerepo "github.com/mihb123/quanly-phongtro/internal/repository/revenue"
	roomrepo "github.com/mihb123/quanly-phongtro/internal/repository/room"
	tenantrepo "github.com/mihb123/quanly-phongtro/internal/repository/tenant"
	httpRouter "github.com/mihb123/quanly-phongtro/internal/router"
	"github.com/mihb123/quanly-phongtro/internal/security"
	authsvc "github.com/mihb123/quanly-phongtro/internal/service/auth"
	"github.com/mihb123/quanly-phongtro/internal/service/email"
	geosvc "github.com/mihb123/quanly-phongtro/internal/service/geo"
	housesvc "github.com/mihb123/quanly-phongtro/internal/service/house"
	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"
	revenuesvc "github.com/mihb123/quanly-phongtro/internal/service/revenue"
	roomsvc "github.com/mihb123/quanly-phongtro/internal/service/room"
	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"
	tenantsvc "github.com/mihb123/quanly-phongtro/internal/service/tenant"
	zalosvc "github.com/mihb123/quanly-phongtro/internal/service/zalo"
	"github.com/mihb123/quanly-phongtro/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	sqlDB, err := db.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer sqlDB.Close()
	emailSender := email.NewGoogleSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.MailFromEmail, cfg.MailFromName)
	verifyEmailRepo := authrepo.NewEmailVerificationRepository(sqlDB)
	otpCheckRepo := authrepo.NewOTPCheckRepository(sqlDB)
	jwtRepo := authrepo.NewAuthSessionRepository(sqlDB)
	userRepo := authrepo.NewUserRepository(sqlDB)
	hasher := security.NewBcryptHasher()
	tokenProvider := security.NewJWTProvider(cfg.AccessTokenJWTSecret, cfg.RefreshTokenJWTSecret, cfg.TokenTTL, jwtRepo)
	geoIPService := geosvc.NewGeoIPServiceFromBytes(assets.GeoLite2City)
	defer geoIPService.Close()
	geocodingService := geosvc.NewGeocodingService(cfg.GoogleMapAPIKey)
	authService := authsvc.NewAuthService(userRepo, hasher, tokenProvider, verifyEmailRepo, emailSender, cfg.OTPEXpireMinutes, otpCheckRepo, geoIPService, geocodingService)

	houseCostRepo := houserepo.NewHouseCostRepository(sqlDB)
	revenueSummaryRepo := revenuerepo.NewRevenueSummaryRepository(sqlDB)

	revenueWorker := revenuesvc.NewRevenueWorker(revenueSummaryRepo, houseCostRepo)
	revenueWorker.Start()
	// defer revenueWorker.Stop() // will stop before shutdown

	eventBus := revenuesvc.NewEventBus()
	eventBus.Subscribe(revenuesvc.EventInvoiceChanged, func(payload interface{}) {
		if p, ok := payload.(revenuesvc.RevenueSummaryPayload); ok {
			revenueWorker.Enqueue(p.HouseID, p.Period)
		}
	})
	eventBus.Subscribe(revenuesvc.EventHouseCostChanged, func(payload interface{}) {
		if p, ok := payload.(revenuesvc.RevenueSummaryPayload); ok {
			revenueWorker.Enqueue(p.HouseID, p.Period)
		}
	})

	houseRepo := houserepo.NewHouseRepository(sqlDB)
	houseService := housesvc.NewHouseServiceImpt(houseRepo, houseCostRepo)
	invoiceRepo := invoicerepo.NewInvoiceRepository(sqlDB)

	roomRepo := roomrepo.NewRoomRepository(sqlDB)
	roomService := roomsvc.NewRoomService(roomRepo, houseRepo)

	tenantRepo := tenantrepo.NewTenantRepository(sqlDB)
	tenantService := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, houseRepo, hasher)
	tenanHandler := tenanthandler.NewTenantHandler(tenantService)

	imageService := invoicesvc.NewImageService()
	invoiceService := invoicesvc.NewInvoiceService(invoiceRepo, roomRepo, houseRepo, tenantRepo, eventBus)
	invoiceHandler := invoicehandler.NewInvoiceHandler(invoiceService, imageService)

	houseHandler := househandler.NewHouseHandler(houseService, invoiceService)
	roomHandler := roomhandler.NewRoomHandler(roomService, invoiceService)

	houseCostService := housesvc.NewHouseCostService(houseCostRepo, houseRepo, eventBus, revenueSummaryRepo)
	houseCostHandler := househandler.NewHouseCostHandler(houseCostService)

	zaloClient := zalosvc.NewZaloClient()
	webhookBaseURL := "https://" + cfg.AppURL
	if cfg.AppEnv == "dev" && cfg.AppURLDev != "" {
		webhookBaseURL = "https://" + cfg.AppURLDev
		if cfg.AppPortDev != "" {
			webhookBaseURL += ":" + cfg.AppPortDev
		}
	}
	authHandler := authhandler.NewAuthHandler(
		authService,
		authhandler.WithSecureCookies(cfg.CookieSecure),
		authhandler.WithTrustedProxies(cfg.TrustedProxyCIDRs),
	)

	invoicePaymentRepo := invoicerepo.NewInvoicePaymentRepository(sqlDB)
	appSecretKeyBytes, err := sharedsvc.DecodeAES256Key(cfg.AppSecretEncryptionKey, "APP_SECRET_ENCRYPTION_KEY")
	if err != nil {
		log.Fatalf("decode app secret encryption key: %v", err)
	}
	paymentCredentialService := paymentsvc.NewPaymentCredentialService(invoicePaymentRepo, appSecretKeyBytes, paymentsvc.PayOSCredentials{
		ClientID:    cfg.PayOSClientID,
		APIKey:      cfg.PayOSApiKey,
		ChecksumKey: cfg.PayOSChecksumKey,
	})
	paymentRegistry := paymentsvc.NewPaymentProviderRegistry(paymentsvc.NewPayOSProvider(), paymentsvc.NewSePayProvider())
	paymentService := paymentsvc.NewPaymentService(invoicePaymentRepo, invoiceRepo, tenantRepo, userRepo, nil, paymentCredentialService, paymentRegistry, webhookBaseURL)

	zaloService, err := zalosvc.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, invoiceRepo, imageService, paymentService, cfg.ZaloBotEncryptionKey, webhookBaseURL, cfg.UploadURLSigningKey)
	if err != nil {
		log.Fatalf("failed to init zalo service: %v", err)
	}
	if configurablePaymentService, ok := paymentService.(interface {
		SetZaloService(paymentsvc.ZaloMessenger)
	}); ok {
		configurablePaymentService.SetZaloService(zaloService)
	}
	keyBytes, err := sharedsvc.DecodeEncryptionKey(cfg.ZaloBotEncryptionKey)
	if err != nil {
		log.Fatalf("decode zalo encryption key: %v", err)
	}
	pendingInvoiceUpdateRepo := invoicerepo.NewPendingInvoiceUpdateRepository(sqlDB)
	zaloInvoiceCommandService := zalosvc.NewZaloInvoiceCommandService(invoiceService, invoiceRepo, roomRepo, houseRepo, tenantRepo, userRepo, pendingInvoiceUpdateRepo, zaloClient, imageService, paymentService, keyBytes, webhookBaseURL)
	if configurableZaloService, ok := zaloService.(interface {
		SetInvoiceCommandService(zalosvc.ZaloInvoiceCommandService)
	}); ok {
		configurableZaloService.SetInvoiceCommandService(zaloInvoiceCommandService)
	}
	zaloHandler := zalohandler.NewZaloHandler(zaloService, webhookBaseURL)

	// Start the Zalo token health check cron (every 4 hours)
	zaloCron := zalosvc.NewZaloCronService(zaloClient, userRepo, keyBytes)
	zaloCron.Start()
	// Trigger an immediate check on startup to quickly detect stale tokens
	go zaloCron.RunNow()

	paymentHandler := paymenthandler.NewPaymentHandler(paymentService, paymentCredentialService, webhookBaseURL)
	sePayReconciliationService := paymentsvc.NewSePayReconciliationService(paymentsvc.NewSePayClient(), paymentCredentialService, paymentService)
	paymentHandler.SetSePayReconciler(sePayReconciliationService)

	frontendFS, err := web.DistFS()
	if err != nil {
		log.Fatalf("load embedded frontend: %v", err)
	}

	router := httpRouter.New(
		authHandler,
		houseHandler,
		roomHandler,
		tokenProvider,
		tenanHandler,
		invoiceHandler,
		zaloHandler,
		houseCostHandler,
		paymentHandler,
		httpRouter.WithTrustedProxies(cfg.TrustedProxyCIDRs),
		httpRouter.WithUploadURLSigningKey(cfg.UploadURLSigningKey),
		httpRouter.WithStaticFS(frontendFS),
	)
	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on http://localhost:%s", cfg.AppPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("start server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}

	revenueWorker.Stop()
	zaloCron.Stop()
}
