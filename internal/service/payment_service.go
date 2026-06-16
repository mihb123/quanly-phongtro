package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

type PaymentService interface {
	CreatePaymentLinkForInvoice(ctx context.Context, managerID, provider string, invoice *model.InvoiceWithRoom, tenantName string) (*model.InvoicePaymentLink, error)
	CancelPaymentLink(ctx context.Context, managerID, provider, providerOrderRef, reason string) error
	HandleWebhook(ctx context.Context, provider, managerID string, body []byte) error
}

type paymentRepository interface {
	CreatePaymentLink(ctx context.Context, link *model.InvoicePaymentLink) error
	GetActivePaymentLinkByProvider(ctx context.Context, invoiceID, provider string) (*model.InvoicePaymentLink, error)
	GetPaymentLinkByProviderOrderRef(ctx context.Context, provider, providerOrderRef string) (*model.InvoicePaymentLink, error)
	GetPaymentLinkByProviderOrderRefForManager(ctx context.Context, managerID, provider, providerOrderRef string) (*model.InvoicePaymentLink, error)
	UpdatePaymentLinkStatus(ctx context.Context, id, status string) error
	CheckProviderEventExists(ctx context.Context, provider, providerOrderRef, transactionRef string) (bool, error)
	CreatePaymentEvent(ctx context.Context, event *model.PaymentEvent) error
}

type paymentInvoiceRepository interface {
	SystemUpdateInvoiceStatusAndMethod(ctx context.Context, id, status, method string) (*model.InvoiceWithRoom, error)
}

type paymentTenantRepository interface {
	ListTenantByRoomID(ctx context.Context, managerID, roomID string) ([]model.FullInfoTenant, error)
}

type paymentUserRepository interface {
	GetByUserID(ctx context.Context, userID string) (*model.User, error)
}

type paymentService struct {
	paymentRepo       paymentRepository
	invoiceRepo       paymentInvoiceRepository
	tenantRepo        paymentTenantRepository
	userRepo          paymentUserRepository
	zaloService       ZaloService
	credentialService PaymentCredentialService
	registry          *PaymentProviderRegistry
	appURL            string
}

// NewPaymentService wires provider-neutral payment orchestration.
func NewPaymentService(
	paymentRepo paymentRepository,
	invoiceRepo paymentInvoiceRepository,
	tenantRepo paymentTenantRepository,
	userRepo paymentUserRepository,
	zaloService ZaloService,
	credentialService PaymentCredentialService,
	registry *PaymentProviderRegistry,
	appURL string,
) PaymentService {
	return &paymentService{
		paymentRepo:       paymentRepo,
		invoiceRepo:       invoiceRepo,
		tenantRepo:        tenantRepo,
		userRepo:          userRepo,
		zaloService:       zaloService,
		credentialService: credentialService,
		registry:          registry,
		appURL:            strings.TrimRight(appURL, "/"),
	}
}

// SetZaloService attaches optional payment notification delivery after service construction.
func (s *paymentService) SetZaloService(zaloService ZaloService) {
	s.zaloService = zaloService
}

// CreatePaymentLinkForInvoice reuses an active provider link or creates and stores a new one.
func (s *paymentService) CreatePaymentLinkForInvoice(ctx context.Context, managerID, provider string, invoice *model.InvoiceWithRoom, tenantName string) (*model.InvoicePaymentLink, error) {
	provider = normalizePaymentProvider(provider)
	providerAdapter, err := s.provider(provider)
	if err != nil {
		return nil, err
	}

	credentials, err := s.credentialService.GetCredentials(ctx, managerID, provider)
	if err != nil {
		return nil, err
	}

	activeLink, err := s.paymentRepo.GetActivePaymentLinkByProvider(ctx, invoice.ID, provider)
	if err != nil {
		return nil, fmt.Errorf("get active payment link: %w", err)
	}
	if activeLink != nil {
		return activeLink, nil
	}

	providerLink, err := providerAdapter.CreatePaymentLink(ctx, PaymentCreateInput{
		Invoice:     invoice,
		TenantName:  tenantName,
		ManagerID:   managerID,
		AppURL:      s.appURL,
		Credentials: credentials,
		ProviderOrderRefExists: func(ctx context.Context, providerOrderRef string) (bool, error) {
			link, err := s.paymentRepo.GetPaymentLinkByProviderOrderRef(ctx, provider, providerOrderRef)
			return link != nil, err
		},
	})
	if err != nil {
		return nil, err
	}

	paymentLink := &model.InvoicePaymentLink{
		InvoiceID:        invoice.ID,
		Provider:         provider,
		ProviderOrderRef: providerLink.ProviderOrderRef,
		OrderCode:        providerLink.OrderCode,
		PaymentLinkID:    providerLink.PaymentLinkID,
		CheckoutURL:      providerLink.CheckoutURL,
		QRCode:           providerLink.QRCode,
		Amount:           providerLink.Amount,
		Status:           model.PaymentLinkStatusActive,
	}
	if err := s.paymentRepo.CreatePaymentLink(ctx, paymentLink); err != nil {
		return nil, fmt.Errorf("save payment link to DB: %w", err)
	}
	return paymentLink, nil
}

// CancelPaymentLink cancels the remote provider link and marks the local link cancelled.
func (s *paymentService) CancelPaymentLink(ctx context.Context, managerID, provider, providerOrderRef, reason string) error {
	provider = normalizePaymentProvider(provider)
	providerAdapter, err := s.provider(provider)
	if err != nil {
		return err
	}
	credentials, err := s.credentialService.GetCredentials(ctx, managerID, provider)
	if err != nil {
		return err
	}
	if err := providerAdapter.CancelPaymentLink(ctx, PaymentCancelInput{
		Credentials:      credentials,
		ProviderOrderRef: providerOrderRef,
		Reason:           reason,
	}); err != nil {
		return err
	}

	link, err := s.paymentRepo.GetPaymentLinkByProviderOrderRefForManager(ctx, managerID, provider, providerOrderRef)
	if err != nil {
		return err
	}
	if link == nil {
		return nil
	}
	if err := s.paymentRepo.UpdatePaymentLinkStatus(ctx, link.ID, model.PaymentLinkStatusCancelled); err != nil {
		return fmt.Errorf("update cancelled payment link status: %w", err)
	}
	return nil
}

// HandleWebhook verifies, records, and applies a provider webhook event.
func (s *paymentService) HandleWebhook(ctx context.Context, provider, managerID string, body []byte) error {
	provider = normalizePaymentProvider(provider)
	providerAdapter, err := s.provider(provider)
	if err != nil {
		return err
	}
	credentials, err := s.credentialService.GetCredentials(ctx, managerID, provider)
	if err != nil {
		return err
	}
	verifiedEvent, err := providerAdapter.VerifyWebhook(ctx, PaymentWebhookInput{
		Body:        body,
		Credentials: credentials,
	})
	if errors.Is(err, ErrPaymentWebhookIgnored) {
		return nil
	}
	if err != nil {
		log.Printf("Invalid %s webhook signature: %v", provider, err)
		return err
	}
	if verifiedEvent == nil {
		return ErrPayOSVerifiedDataNil
	}

	exists, err := s.paymentRepo.CheckProviderEventExists(ctx, provider, verifiedEvent.ProviderOrderRef, verifiedEvent.TransactionReference)
	if err != nil {
		return fmt.Errorf("check event exists: %w", err)
	}
	if exists {
		return nil
	}

	paymentLink, err := s.paymentRepo.GetPaymentLinkByProviderOrderRefForManager(ctx, managerID, provider, verifiedEvent.ProviderOrderRef)
	if err != nil {
		return fmt.Errorf("get payment link: %w", err)
	}
	invoiceID, matchingMethod := matchedPaymentInvoice(paymentLink)
	event := newPaymentEvent(provider, managerID, invoiceID, verifiedEvent, matchingMethod)
	if err := s.paymentRepo.CreatePaymentEvent(ctx, event); err != nil {
		return fmt.Errorf("create payment event: %w", err)
	}
	if invoiceID == "" {
		log.Printf("Unmatched %s transaction: %s", provider, verifiedEvent.ProviderOrderRef)
		return nil
	}

	return s.processInvoicePayment(ctx, provider, invoiceID, verifiedEvent.Amount, paymentLink)
}

// provider returns a registered provider adapter or a domain error.
func (s *paymentService) provider(provider string) (PaymentProvider, error) {
	providerAdapter, ok := s.registry.Get(provider)
	if !ok {
		return nil, ErrPaymentProviderNotFound
	}
	return providerAdapter, nil
}

// processInvoicePayment marks the invoice paid and sends best-effort notifications.
func (s *paymentService) processInvoicePayment(ctx context.Context, provider, invoiceID string, amount int, paymentLink *model.InvoicePaymentLink) error {
	invoiceWithRoom, err := s.invoiceRepo.SystemUpdateInvoiceStatusAndMethod(ctx, invoiceID, model.InvoiceStatusPaid, paymentMethodForProvider(provider))
	if err != nil {
		return fmt.Errorf("system update invoice status: %w", err)
	}

	managerID := invoiceWithRoom.ManagerID
	if paymentLink != nil {
		if err := s.paymentRepo.UpdatePaymentLinkStatus(ctx, paymentLink.ID, model.PaymentLinkStatusPaid); err != nil {
			log.Printf("failed to mark %s payment link %s paid: %v", provider, paymentLink.ID, err)
		}
	}

	tenants, err := s.tenantRepo.ListTenantByRoomID(ctx, managerID, invoiceWithRoom.RoomID)
	if err != nil {
		log.Printf("failed to list tenants for room %s: %v", invoiceWithRoom.RoomID, err)
	}
	tenantName := defaultTenantName
	if len(tenants) > 0 {
		tenantName = tenants[0].FullName
	}

	tenantMsg := fmt.Sprintf("Hóa đơn phòng %s tháng %s (%s) đã được thanh toán thành công.",
		invoiceWithRoom.RoomName, invoiceWithRoom.Period, formatCurrency(amount))
	for _, tenant := range tenants {
		if tenant.ZaloUserID == "" {
			continue
		}
		s.sendTextMessageBestEffort(ctx, managerID, tenant.ZaloUserID, tenantMsg)
	}

	managerMsg := fmt.Sprintf("[Thanh toán] Khách hàng %s tại phòng %s vừa thanh toán thành công %s. Hóa đơn đã tự động chuyển sang đã thanh toán.",
		tenantName, invoiceWithRoom.RoomName, formatCurrency(amount))
	managerUser, err := s.userRepo.GetByUserID(ctx, managerID)
	if err == nil && managerUser.ZaloUserID != nil && *managerUser.ZaloUserID != "" {
		s.sendTextMessageBestEffort(ctx, managerID, *managerUser.ZaloUserID, managerMsg)
		return nil
	}
	if err != nil {
		log.Printf("failed to fetch manager %s for payment notification: %v", managerID, err)
	}
	log.Printf("Manager Notification: %s", managerMsg)
	return nil
}

// sendTextMessageBestEffort logs notification errors without failing payment processing.
func (s *paymentService) sendTextMessageBestEffort(ctx context.Context, managerID, chatID, text string) {
	if s.zaloService == nil {
		return
	}
	if err := s.zaloService.SendTextMessage(ctx, managerID, chatID, text); err != nil {
		log.Printf("failed to send payment notification to chat %s: %v", chatID, err)
	}
}

// matchedPaymentInvoice resolves the invoice attached to a payment link.
func matchedPaymentInvoice(paymentLink *model.InvoicePaymentLink) (string, string) {
	if paymentLink == nil {
		return "", paymentMatchUnmatched
	}
	return paymentLink.InvoiceID, paymentMatchOrderRef
}

// newPaymentEvent maps verified webhook data into an audit event.
func newPaymentEvent(provider, managerID, invoiceID string, data *VerifiedPaymentEvent, matchingMethod string) *model.PaymentEvent {
	eventStatus := model.PaymentEventStatusUnmatched
	if invoiceID != "" {
		eventStatus = model.PaymentEventStatusProcessed
	}
	managerIDPtr := nullableString(managerID)
	invoiceIDPtr := nullableString(invoiceID)

	return &model.PaymentEvent{
		Provider:             provider,
		ManagerID:            managerIDPtr,
		InvoiceID:            invoiceIDPtr,
		ProviderOrderRef:     data.ProviderOrderRef,
		OrderCode:            data.OrderCode,
		Amount:               data.Amount,
		TransactionReference: data.TransactionReference,
		PayerAccount:         data.PayerAccount,
		CounterAccount:       data.CounterAccount,
		RawPayload:           data.RawPayload,
		SignatureResult:      data.SignatureResult,
		MatchingMethod:       &matchingMethod,
		Status:               eventStatus,
	}
}

// nullableString returns nil for empty strings for optional event columns.
func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// normalizePaymentProvider resolves empty provider names to PayOS for current compatibility.
func normalizePaymentProvider(provider string) string {
	if provider == "" {
		return model.PaymentProviderPayOS
	}
	return strings.ToLower(provider)
}

// paymentMethodForProvider maps provider keys to invoice payment method values.
func paymentMethodForProvider(provider string) string {
	if provider == model.PaymentProviderPayOS {
		return model.PaymentMethodPayOS
	}
	return strings.ToUpper(provider)
}

// formatCurrency returns a simple VND amount string for chat notifications.
func formatCurrency(amount int) string {
	return fmt.Sprintf("%d VND", amount)
}
