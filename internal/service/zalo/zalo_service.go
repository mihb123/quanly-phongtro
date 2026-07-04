package zalo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"
	"github.com/mihb123/quanly-phongtro/internal/service/shared"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type ZaloService interface {
	SaveZaloConfig(ctx context.Context, managerID, botToken, webhookUrl, secretToken string) error
	GetZaloConfigStatus(ctx context.Context, managerID string) (ZaloBotStatus, error)
	SendTextMessage(ctx context.Context, managerID, chatID, text string) error
	HandleWebhook(ctx context.Context, managerID string, body []byte, secretTokenHeader string) error
	SendInvoiceToZalo(ctx context.Context, managerID, invoiceID string) error
}

// ZaloBotStatus describes the current connectivity status of a manager's Zalo bot.
type ZaloBotStatus struct {
	HasConfig bool   `json:"has_config"`
	IsActive  bool   `json:"is_zalo_bot_active"`
	IsLinked  bool   `json:"is_linked"`
	BotID     string `json:"bot_id"`
	ManagerID string `json:"manager_id"`
}

type zaloServiceImpl struct {
	client                ZaloClient
	userRepo              model.UserRepository
	roomRepo              model.RoomRepository
	tenantRepo            model.TenantRepository
	houseRepo             model.HouseRepository
	invoiceRepo           model.InvoiceRepository
	imageService          invoicesvc.ImageService
	paymentService        paymentsvc.PaymentService
	invoiceCommandService ZaloInvoiceCommandService
	encryptionKey         []byte
	publicBaseURL         string
}

// NewZaloService creates the manager-specific Zalo integration service.
func NewZaloService(client ZaloClient, userRepo model.UserRepository, roomRepo model.RoomRepository, tenantRepo model.TenantRepository, houseRepo model.HouseRepository, invoiceRepo model.InvoiceRepository, imageService invoicesvc.ImageService, paymentService paymentsvc.PaymentService, hexKey string, publicBaseURL ...string) (ZaloService, error) {
	keyBytes, err := shared.DecodeEncryptionKey(hexKey)
	if err != nil {
		return nil, err
	}

	baseURL := "http://localhost:8080"
	if len(publicBaseURL) > 0 && publicBaseURL[0] != "" {
		baseURL = strings.TrimRight(publicBaseURL[0], "/")
	}

	return &zaloServiceImpl{
		client:         client,
		userRepo:       userRepo,
		roomRepo:       roomRepo,
		tenantRepo:     tenantRepo,
		houseRepo:      houseRepo,
		invoiceRepo:    invoiceRepo,
		imageService:   imageService,
		paymentService: paymentService,
		encryptionKey:  keyBytes,
		publicBaseURL:  baseURL,
	}, nil
}

// SetInvoiceCommandService attaches the optional invoice chat command handler.
func (s *zaloServiceImpl) SetInvoiceCommandService(commandService ZaloInvoiceCommandService) {
	s.invoiceCommandService = commandService
}

func (s *zaloServiceImpl) SaveZaloConfig(ctx context.Context, managerID, botToken, webhookUrl, secretToken string) error {
	// 1. Validate bot token by calling GetMe
	if _, err := s.client.GetMe(ctx, botToken); err != nil {
		return fmt.Errorf("invalid bot token: %v", err)
	}

	// 2. Set Webhook
	if err := s.client.SetWebhook(ctx, botToken, webhookUrl, secretToken); err != nil {
		return fmt.Errorf("set webhook failed: %v", err)
	}

	// 3. Encrypt token and webhook secret
	encToken, err := security.Encrypt(botToken, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt token: %v", err)
	}
	encSecret, err := security.Encrypt(secretToken, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt webhook secret: %v", err)
	}

	// 4. Save to DB and mark bot as active
	active := true
	input := model.UpdateUserInput{
		ZaloBotToken:      &encToken,
		ZaloWebhookSecret: &encSecret,
		IsZaloBotActive:   &active,
	}

	_, err = s.userRepo.UpdateUser(ctx, managerID, input)
	return err
}

// GetZaloConfigStatus returns the current Zalo bot configuration and token health for a manager.
func (s *zaloServiceImpl) GetZaloConfigStatus(ctx context.Context, managerID string) (ZaloBotStatus, error) {
	user, err := s.userRepo.GetByUserID(ctx, managerID)
	if err != nil {
		return ZaloBotStatus{}, err
	}

	hasConfig := user.ZaloBotToken != nil && *user.ZaloBotToken != ""
	status := ZaloBotStatus{
		HasConfig: hasConfig,
		IsActive:  hasConfig && user.IsZaloBotActive,
		IsLinked:  user.ZaloUserID != nil && *user.ZaloUserID != "",
		ManagerID: managerID,
	}

	if hasConfig {
		botToken, err := security.Decrypt(*user.ZaloBotToken, s.encryptionKey)
		if err == nil {
			botInfo, err := s.client.GetMe(ctx, botToken)
			if err == nil && botInfo != nil {
				status.BotID = botInfo.AppID
			}
		}
	}

	return status, nil
}

func (s *zaloServiceImpl) getDecryptedToken(ctx context.Context, managerID string) (string, error) {
	user, err := s.userRepo.GetByUserID(ctx, managerID)
	if err != nil {
		return "", err
	}

	if user.ZaloBotToken == nil || *user.ZaloBotToken == "" {
		return "", errors.New("zalo config not found for manager")
	}

	botToken, err := security.Decrypt(*user.ZaloBotToken, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt bot token failed: %v", err)
	}

	return botToken, nil
}

// markTokenInactive marks the manager's Zalo bot token as inactive in the DB.
// Called when an API call indicates the token is no longer valid.
func (s *zaloServiceImpl) markTokenInactive(ctx context.Context, managerID string) {
	inactive := false
	_, _ = s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{IsZaloBotActive: &inactive})
}

func (s *zaloServiceImpl) SendTextMessage(ctx context.Context, managerID, chatID, text string) error {
	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err != nil {
		return err
	}

	return s.client.SendMessage(ctx, botToken, chatID, text)
}

func (s *zaloServiceImpl) SendInvoiceToZalo(ctx context.Context, managerID, invoiceID string) error {
	return deliverInvoiceToZalo(ctx, zaloInvoiceDeliveryDeps{
		client:            s.client,
		userRepo:          s.userRepo,
		roomRepo:          s.roomRepo,
		tenantRepo:        s.tenantRepo,
		invoiceRepo:       s.invoiceRepo,
		imageService:      s.imageService,
		paymentService:    s.paymentService,
		encryptionKey:     s.encryptionKey,
		publicBaseURL:     s.publicBaseURL,
		markTokenInactive: s.markTokenInactive,
	}, managerID, invoiceID)
}
