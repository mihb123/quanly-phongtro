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
	"github.com/mihb123/quanly-phongtro/internal/db"
	httpHandler "github.com/mihb123/quanly-phongtro/internal/handler"
	"github.com/mihb123/quanly-phongtro/internal/repository"
	httpRouter "github.com/mihb123/quanly-phongtro/internal/router"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/mihb123/quanly-phongtro/internal/service/email"
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
	verifyEmailRepo := repository.NewEmailVerificationRepository(sqlDB)
	otpCheckRepo := repository.NewOTPCheckRepository(sqlDB)
	jwtRepo := repository.NewAuthSessionRepository(sqlDB)
	userRepo := repository.NewUserRepository(sqlDB)
	hasher := security.NewBcryptHasher()
	tokenProvider := security.NewJWTProvider(cfg.AccessTokenJWTSecret, cfg.RefreshTokenJWTSecret, cfg.TokenTTL, jwtRepo)
	geoIPService := service.NewGeoIPService("internal/assets/geoip/GeoLite2-City.mmdb")
	defer geoIPService.Close()
	geocodingService := service.NewGeocodingService(cfg.GoogleMapAPIKey)
	authService := service.NewAuthService(userRepo, hasher, tokenProvider, verifyEmailRepo, emailSender, cfg.OTPEXpireMinutes, otpCheckRepo, geoIPService, geocodingService)
	authHandler := httpHandler.NewAuthHandler(authService)

	houseRepo := repository.NewHouseRepository(sqlDB)
	houseService := service.NewHouseServiceImpt(houseRepo)
	invoiceRepo := repository.NewInvoiceRepository(sqlDB)
	
	roomRepo := repository.NewRoomRepository(sqlDB)
	roomService := service.NewRoomService(roomRepo, houseRepo)

	tenantRepo := repository.NewTenantRepository(sqlDB)
	tenantService := service.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, hasher)
	tenanHandler := httpHandler.NewTenantHandler(tenantService)

	imageService := service.NewImageService()
	invoiceService := service.NewInvoiceService(invoiceRepo, roomRepo, houseRepo, tenantRepo)
	invoiceHandler := httpHandler.NewInvoiceHandler(invoiceService, imageService)

	houseHandler := httpHandler.NewHouseHandler(houseService, invoiceService)
	roomHandler := httpHandler.NewRoomHandler(roomService, invoiceService)

	zaloClient := service.NewZaloClient()
	zaloService, err := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, invoiceRepo, imageService, cfg.ZaloBotEncryptionKey)
	if err != nil {
		log.Fatalf("failed to init zalo service: %v", err)
	}
	webhookBaseURL := "https://" + cfg.AppURL
	if cfg.AppEnv == "dev" && cfg.AppURLDev != "" {
		webhookBaseURL = "https://" + cfg.AppURLDev
	}
	zaloHandler := httpHandler.NewZaloHandler(zaloService, webhookBaseURL)

	// Start the Zalo token health check cron (every 4 hours)
	keyBytes, _ := service.DecodeEncryptionKey(cfg.ZaloBotEncryptionKey)
	zaloCron := service.NewZaloCronService(zaloClient, userRepo, keyBytes)
	zaloCron.Start()
	// Trigger an immediate check on startup to quickly detect stale tokens
	go zaloCron.RunNow()

	router := httpRouter.New(authHandler, houseHandler, roomHandler, tokenProvider, tenanHandler, invoiceHandler, zaloHandler)
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

	zaloCron.Stop()
}
