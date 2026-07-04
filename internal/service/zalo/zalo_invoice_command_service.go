package zalo

import (
	"context"
	"errors"
	"strings"

	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type zaloInvoiceCommandServiceImpl struct {
	invoiceService invoicesvc.InvoiceService
	invoiceRepo    model.InvoiceRepository
	roomRepo       model.RoomRepository
	houseRepo      model.HouseRepository
	tenantRepo     model.TenantRepository
	userRepo       model.UserRepository
	pendingRepo    model.PendingInvoiceUpdateRepository
	zaloClient     ZaloClient
	imageService   invoicesvc.ImageService
	paymentService paymentsvc.PaymentService
	encryptionKey  []byte
	publicBaseURL  string
}

// NewZaloInvoiceCommandService creates the business handler for invoice chat commands.
func NewZaloInvoiceCommandService(invoiceService invoicesvc.InvoiceService, invoiceRepo model.InvoiceRepository, roomRepo model.RoomRepository, houseRepo model.HouseRepository, tenantRepo model.TenantRepository, userRepo model.UserRepository, pendingRepo model.PendingInvoiceUpdateRepository, zaloClient ZaloClient, imageService invoicesvc.ImageService, paymentService paymentsvc.PaymentService, encryptionKey []byte, publicBaseURL string) ZaloInvoiceCommandService {
	return &zaloInvoiceCommandServiceImpl{
		invoiceService: invoiceService,
		invoiceRepo:    invoiceRepo,
		roomRepo:       roomRepo,
		houseRepo:      houseRepo,
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		pendingRepo:    pendingRepo,
		zaloClient:     zaloClient,
		imageService:   imageService,
		paymentService: paymentService,
		encryptionKey:  encryptionKey,
		publicBaseURL:  strings.TrimRight(publicBaseURL, "/"),
	}
}

// HandleInvoiceCommand processes a parsed invoice command from a Zalo webhook message.
func (s *zaloInvoiceCommandServiceImpl) HandleInvoiceCommand(ctx context.Context, managerID string, webhookCtx webhookMessageContext) error {
	parsed := ParseCommand(webhookCtx.text)
	chatID := commandChatID(webhookCtx)
	if chatID == "" {
		return nil
	}

	pending, err := s.pendingRepo.GetByChatID(ctx, managerID, chatID)
	if err != nil && !errors.Is(err, model.ErrPendingInvoiceUpdateNotFound) {
		return err
	}
	if pending != nil {
		handled, err := s.handlePendingCommand(ctx, managerID, chatID, webhookCtx, parsed, pending)
		if handled || err != nil {
			return err
		}
	}

	switch parsed.Type {
	case CommandUtilitySingle:
		return s.handleSingleCommand(ctx, managerID, webhookCtx, parsed, "", false)
	case CommandUtilityBatch:
		return s.handleBatchCommand(ctx, managerID, webhookCtx, parsed, "", false)
	case CommandConfirm, CommandCancel, CommandPeriodSelect:
		return s.sendTextMessage(ctx, managerID, chatID, "Không có thao tác hóa đơn nào đang chờ xử lý.")
	default:
		return nil
	}
}
