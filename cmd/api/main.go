package main

import (
	"context"
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
	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/logging/filelog"
	"github.com/mihb123/quanly-phongtro/internal/plugin"
	authrepo "github.com/mihb123/quanly-phongtro/internal/repository/auth"
	houserepo "github.com/mihb123/quanly-phongtro/internal/repository/house"
	invoicerepo "github.com/mihb123/quanly-phongtro/internal/repository/invoice"
	jobrepo "github.com/mihb123/quanly-phongtro/internal/repository/jobs"
	pluginrepo "github.com/mihb123/quanly-phongtro/internal/repository/plugin"
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
	jobsvc "github.com/mihb123/quanly-phongtro/internal/service/jobs"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"
	roomsvc "github.com/mihb123/quanly-phongtro/internal/service/room"
	sharedsvc "github.com/mihb123/quanly-phongtro/internal/service/shared"
	tenantsvc "github.com/mihb123/quanly-phongtro/internal/service/tenant"
	thuchisvc "github.com/mihb123/quanly-phongtro/internal/service/thuchi"
	zalosvc "github.com/mihb123/quanly-phongtro/internal/service/zalo"
	"github.com/mihb123/quanly-phongtro/internal/web"
	"github.com/mihb123/quanly-phongtro/pkg/sepay"
	"github.com/mihb123/quanly-phongtro/pkg/zalobot"
	_ "github.com/mihb123/quanly-phongtro/plugins"
	thuchiplugin "github.com/mihb123/quanly-phongtro/plugins/thuchi"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		// Logger chưa có cấu hình: dùng production logger mặc định để vẫn ra JSON.
		zap.Must(logging.New("prod", "info")).Fatal("load config", zap.Error(err))
	}

	zl, err := logging.New(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
	defer func() { _ = zl.Sync() }()
	zap.ReplaceGlobals(zl)

	sqlDB, err := db.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		zl.Fatal("connect database", zap.Error(err))
	}
	defer sqlDB.Close()
	emailSender := email.NewGoogleSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.MailFromEmail, cfg.MailFromName)
	verifyEmailRepo := authrepo.NewEmailVerificationRepository(sqlDB)
	otpCheckRepo := authrepo.NewOTPCheckRepository(sqlDB)
	jwtRepo := authrepo.NewAuthSessionRepository(sqlDB)
	userRepo := authrepo.NewUserRepository(sqlDB)
	hasher := security.NewBcryptHasher()
	tokenProvider := security.NewJWTProvider(cfg.AccessTokenJWTSecret, cfg.RefreshTokenJWTSecret, cfg.TokenTTL, jwtRepo)
	geoIPService, err := geosvc.NewGeoIPServiceFromBytes(assets.GeoLite2City)
	if err != nil {
		zl.Warn("geoip disabled", zap.Error(err))
	}
	defer geoIPService.Close()
	geocodingService := geosvc.NewGeocodingService(cfg.GoogleMapAPIKey)
	authService := authsvc.NewAuthService(userRepo, hasher, tokenProvider, verifyEmailRepo, emailSender, cfg.OTPEXpireMinutes, otpCheckRepo, geoIPService, geocodingService)

	houseCostRepo := houserepo.NewHouseCostRepository(sqlDB)
	revenueSummaryRepo := revenuerepo.NewRevenueSummaryRepository(sqlDB)

	jobRepo := jobrepo.NewRepository(sqlDB)
	revenueWorker := jobsvc.NewWorker(jobRepo, "revenue", revenueSummaryRepo.RecalculateJob)
	revenueWorker.Start()

	houseRepo := houserepo.NewHouseRepository(sqlDB)
	invoiceRepo := invoicerepo.NewInvoiceRepository(sqlDB)

	roomRepo := roomrepo.NewRoomRepository(sqlDB)

	tenantRepo := tenantrepo.NewTenantRepository(sqlDB)
	tenantService := tenantsvc.NewTenantServiceImpl(userRepo, tenantRepo, roomRepo, houseRepo, hasher)
	tenanHandler := tenanthandler.NewTenantHandler(tenantService)

	imageService := invoicesvc.NewImageService()
	invoiceService := invoicesvc.NewInvoiceService(invoiceRepo, roomRepo, houseRepo, tenantRepo, nil)
	transaction := func(ctx context.Context, fn func(context.Context) error) error {
		return db.WithinTransaction(ctx, sqlDB, fn)
	}
	roomService := roomsvc.NewRoomService(roomRepo, houseRepo, roomsvc.WithInvoiceRecalculation(invoiceService, transaction))
	houseService := housesvc.NewHouseServiceImpt(houseRepo, houseCostRepo, housesvc.WithInvoiceRecalculation(invoiceService, transaction))
	invoiceHandler := invoicehandler.NewInvoiceHandler(invoiceService, imageService)

	houseHandler := househandler.NewHouseHandler(houseService)
	roomHandler := roomhandler.NewRoomHandler(roomService)

	houseCostService := housesvc.NewHouseCostService(houseCostRepo, houseRepo, nil, revenueSummaryRepo)
	houseCostHandler := househandler.NewHouseCostHandler(houseCostService)

	// Auto-create each house's cost record on the 1st of every month
	houseCostCron := housesvc.NewHouseCostCronService(houseCostService)
	houseCostCron.Start()

	zaloClient := zalobot.NewClient(nil)
	webhookBaseURL := "https://" + cfg.AppURL
	if cfg.AppEnv == "dev" && cfg.AppURLDev != "" {
		webhookBaseURL = "https://" + cfg.AppURLDev
	}
	listenPort := cfg.AppPort
	if cfg.AppEnv == "dev" && cfg.AppPortDev != "" {
		listenPort = cfg.AppPortDev
	}
	authHandler := authhandler.NewAuthHandler(
		authService,
		authhandler.WithSecureCookies(cfg.CookieSecure),
		authhandler.WithTrustedProxies(cfg.TrustedProxyCIDRs),
	)

	invoicePaymentRepo := invoicerepo.NewInvoicePaymentRepository(sqlDB)
	appSecretKeyBytes, err := sharedsvc.DecodeAES256Key(cfg.AppSecretEncryptionKey, "APP_SECRET_ENCRYPTION_KEY")
	if err != nil {
		zl.Fatal("decode app secret encryption key", zap.Error(err))
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
		zl.Fatal("failed to init zalo service", zap.Error(err))
	}
	paymentService.SetZaloService(zaloService)
	notificationWorker := jobsvc.NewWorker(jobRepo, "payment_notification", paymentService.DeliverNotification)
	notificationWorker.Start()
	keyBytes, err := sharedsvc.DecodeEncryptionKey(cfg.ZaloBotEncryptionKey)
	if err != nil {
		zl.Fatal("decode zalo encryption key", zap.Error(err))
	}
	pendingInvoiceUpdateRepo := invoicerepo.NewPendingInvoiceUpdateRepository(sqlDB)
	zaloInvoiceCommandService := zalosvc.NewZaloInvoiceCommandService(invoiceService, invoiceRepo, roomRepo, houseRepo, tenantRepo, tenantService, userRepo, pendingInvoiceUpdateRepo, zaloClient, imageService, paymentService, keyBytes, webhookBaseURL)
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
	sePayReconciliationService := paymentsvc.NewSePayReconciliationService(sepay.NewClient(nil), paymentCredentialService, paymentService)
	paymentHandler.SetSePayReconciler(sePayReconciliationService)

	thuchiTelegram := thuchisvc.TelegramConfig{BotToken: cfg.ThuChiTelegramToken, ChatIDs: cfg.ThuChiTelegramChatIDs, APIBase: cfg.ThuChiTelegramAPIBase}
	if cfg.ThuChiTelegramWebhook {
		thuchiTelegram.WebhookURL = webhookBaseURL + thuchisvc.TelegramWebhookURL
	}
	thuchiModule := thuchisvc.New(sqlDB.DB, appSecretKeyBytes, webhookBaseURL, thuchiTelegram)
	thuchiplugin.Use(thuchiModule.API, thuchiModule.Public, thuchiModule.Hook, thuchiModule.Telegram)

	pluginManager := plugin.NewManager(pluginrepo.NewPluginRepository(sqlDB), appSecretKeyBytes, plugin.Registered())
	if err := pluginManager.EnableInstall(context.Background(), cfg.PluginDir, cfg.PluginAdminEmails); err != nil {
		zl.Warn("plugin install disabled", zap.Error(err))
	}
	defer pluginManager.Close(context.Background())
	thuchiCtx, stopThuchi := context.WithCancel(context.Background())
	defer stopThuchi()
	thuchiModule.Start(thuchiCtx)

	slowAPILogger, err := filelog.NewSlowAPILogger(filelog.SlowAPIConfig{
		Threshold: cfg.SlowAPIThreshold,
		FilePath:  cfg.SlowAPILogFile,
		MaxDays:   cfg.SlowAPILogMaxDays,
	})
	if err != nil {
		zl.Fatal("init slow api logger", zap.Error(err))
	}
	defer slowAPILogger.Close()

	securityLogger, err := filelog.NewSecurityLogger(filelog.SecurityConfig{})
	if err != nil {
		zl.Fatal("init security logger", zap.Error(err))
	}
	defer securityLogger.Close()

	frontendFS, err := web.DistFS()
	if err != nil {
		zl.Fatal("load embedded frontend", zap.Error(err))
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
		httpRouter.WithLogger(zl),
		httpRouter.WithTrustedProxies(cfg.TrustedProxyCIDRs),
		httpRouter.WithUploadURLSigningKey(cfg.UploadURLSigningKey),
		httpRouter.WithStaticFS(frontendFS),
		httpRouter.WithSlowAPILogger(slowAPILogger),
		httpRouter.WithSecurityLogger(securityLogger),
		httpRouter.WithPluginRoutes(pluginManager.Routes),
		httpRouter.WithPluginPublicRoutes(pluginManager.PublicRoutes),
	)
	server := &http.Server{
		Addr:              ":" + listenPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		zl.Info("api listening", zap.String("addr", "http://localhost:"+listenPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zl.Fatal("start server", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		zl.Error("shutdown error", zap.Error(err))
	}

	stopThuchi()
	notificationWorker.Stop()
	revenueWorker.Stop()
	zaloCron.Stop()
	houseCostCron.Stop()
}
