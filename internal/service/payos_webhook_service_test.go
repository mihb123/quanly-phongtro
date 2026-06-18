package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/payOSHQ/payos-lib-golang/v2"
)

type fakePayOSPaymentRepository struct {
	updatedID     string
	updatedStatus string
	event         *model.PaymentEvent
	paymentLink   *model.InvoicePaymentLink
	linkManagerID string
}

// CheckProviderEventExists returns false for tests that do not exercise idempotency.
func (r *fakePayOSPaymentRepository) CheckProviderEventExists(context.Context, string, string, string) (bool, error) {
	return false, nil
}

// CreatePaymentEvent accepts audit events for tests that focus on payment processing.
func (r *fakePayOSPaymentRepository) CreatePaymentEvent(_ context.Context, event *model.PaymentEvent) error {
	r.event = event
	return nil
}

// CreatePaymentLink is unused by webhook tests.
func (r *fakePayOSPaymentRepository) CreatePaymentLink(context.Context, *model.InvoicePaymentLink) error {
	return nil
}

// GetActivePaymentLinkByProvider is unused by webhook tests.
func (r *fakePayOSPaymentRepository) GetActivePaymentLinkByProvider(context.Context, string, string) (*model.InvoicePaymentLink, error) {
	return nil, nil
}

// GetPaymentLinkByProviderOrderRef returns the configured payment link for webhook matching.
func (r *fakePayOSPaymentRepository) GetPaymentLinkByProviderOrderRef(context.Context, string, string) (*model.InvoicePaymentLink, error) {
	return r.paymentLink, nil
}

// GetPaymentLinkByProviderOrderRefForManager returns the configured link only when it belongs to the manager.
func (r *fakePayOSPaymentRepository) GetPaymentLinkByProviderOrderRefForManager(_ context.Context, managerID, provider, providerOrderRef string) (*model.InvoicePaymentLink, error) {
	if r.linkManagerID != "" && r.linkManagerID != managerID {
		return nil, nil
	}
	return r.GetPaymentLinkByProviderOrderRef(context.Background(), provider, providerOrderRef)
}

// UpdatePaymentLinkStatus records the requested payment link transition.
func (r *fakePayOSPaymentRepository) UpdatePaymentLinkStatus(_ context.Context, id, status string) error {
	r.updatedID = id
	r.updatedStatus = status
	return nil
}

// MarkActivePaymentLinksStaleByManager is unused by webhook tests.
func (r *fakePayOSPaymentRepository) MarkActivePaymentLinksStaleByManager(context.Context, string, string) error {
	return nil
}

type fakePayOSInvoiceRepository struct {
	updatedID     string
	updatedStatus string
	updatedMethod string
	invoice       *model.InvoiceWithRoom
}

// SystemUpdateInvoiceStatusAndMethod records the system invoice update request.
func (r *fakePayOSInvoiceRepository) SystemUpdateInvoiceStatusAndMethod(_ context.Context, id, status, method string) (*model.InvoiceWithRoom, error) {
	r.updatedID = id
	r.updatedStatus = status
	r.updatedMethod = method
	return r.invoice, nil
}

type fakePayOSTenantRepository struct {
	managerID string
	roomID    string
	tenants   []model.FullInfoTenant
}

// ListTenantByRoomID records the room lookup and returns configured tenants.
func (r *fakePayOSTenantRepository) ListTenantByRoomID(_ context.Context, managerID, roomID string) ([]model.FullInfoTenant, error) {
	r.managerID = managerID
	r.roomID = roomID
	return r.tenants, nil
}

type fakePayOSUserRepository struct {
	userID string
	user   *model.User
}

// GetByUserID records the manager lookup and returns the configured user.
func (r *fakePayOSUserRepository) GetByUserID(_ context.Context, userID string) (*model.User, error) {
	r.userID = userID
	return r.user, nil
}

type sentPayOSMessage struct {
	managerID string
	chatID    string
	text      string
}

type fakePayOSZaloService struct {
	sent []sentPayOSMessage
}

// SaveZaloConfig is unused by PayOS webhook tests.
func (s *fakePayOSZaloService) SaveZaloConfig(context.Context, string, string, string, string) error {
	return nil
}

// GetZaloConfigStatus is unused by PayOS webhook tests.
func (s *fakePayOSZaloService) GetZaloConfigStatus(context.Context, string) (ZaloBotStatus, error) {
	return ZaloBotStatus{}, nil
}

// SendTextMessage records notification messages sent by payment processing.
func (s *fakePayOSZaloService) SendTextMessage(_ context.Context, managerID, chatID, text string) error {
	s.sent = append(s.sent, sentPayOSMessage{
		managerID: managerID,
		chatID:    chatID,
		text:      text,
	})
	return nil
}

// HandleWebhook is unused by PayOS webhook tests.
func (s *fakePayOSZaloService) HandleWebhook(context.Context, string, []byte, string) error {
	return nil
}

// SendInvoiceToZalo is unused by PayOS webhook tests.
func (s *fakePayOSZaloService) SendInvoiceToZalo(context.Context, string, string) error {
	return nil
}

type fakePaymentCredentialService struct {
	credentials map[string]string
}

// GetCredentials returns configured PayOS credentials for webhook verification.
func (s fakePaymentCredentialService) GetCredentials(context.Context, string, string) (map[string]string, error) {
	return s.credentials, nil
}

// GetPayOSConfig is unused by payment service tests.
func (s fakePaymentCredentialService) GetPayOSConfig(context.Context, string, string) (PayOSConfigStatus, error) {
	return PayOSConfigStatus{}, nil
}

// SavePayOSConfig is unused by payment service tests.
func (s fakePaymentCredentialService) SavePayOSConfig(context.Context, string, PayOSCredentials) error {
	return nil
}

// DeletePayOSConfig is unused by payment service tests.
func (s fakePaymentCredentialService) DeletePayOSConfig(context.Context, string) error {
	return nil
}

// TestPayOSWebhookServiceProcessInvoicePayment verifies a matched PayOS payment marks the invoice paid.
func TestPayOSWebhookServiceProcessInvoicePayment(t *testing.T) {
	managerZaloID := "manager-zalo"
	paymentRepository := &fakePayOSPaymentRepository{}
	invoiceRepository := &fakePayOSInvoiceRepository{
		invoice: &model.InvoiceWithRoom{
			Invoice: model.Invoice{
				ID:     "invoice-1",
				RoomID: "room-1",
				Period: "2024-01",
			},
			RoomName:  "A101",
			ManagerID: "manager-1",
		},
	}
	tenantRepository := &fakePayOSTenantRepository{
		tenants: []model.FullInfoTenant{
			{FullName: "Nguyen Van A", ZaloUserID: "tenant-zalo"},
		},
	}
	userRepository := &fakePayOSUserRepository{
		user: &model.User{ID: "manager-1", ZaloUserID: &managerZaloID},
	}
	zaloService := &fakePayOSZaloService{}
	svc := &paymentService{
		paymentRepo: paymentRepository,
		invoiceRepo: invoiceRepository,
		tenantRepo:  tenantRepository,
		userRepo:    userRepository,
		zaloService: zaloService,
	}

	err := svc.processInvoicePayment(
		context.Background(),
		model.PaymentProviderPayOS,
		"invoice-1",
		150000,
		&model.InvoicePaymentLink{ID: "link-1"},
	)
	if err != nil {
		t.Fatalf("processInvoicePayment() error = %v", err)
	}

	if invoiceRepository.updatedID != "invoice-1" {
		t.Fatalf("updated invoice ID = %q, want invoice-1", invoiceRepository.updatedID)
	}
	if invoiceRepository.updatedStatus != model.InvoiceStatusPaid {
		t.Fatalf("updated invoice status = %q, want %q", invoiceRepository.updatedStatus, model.InvoiceStatusPaid)
	}
	if invoiceRepository.updatedMethod != model.PaymentMethodPayOS {
		t.Fatalf("updated invoice method = %q, want %q", invoiceRepository.updatedMethod, model.PaymentMethodPayOS)
	}
	if paymentRepository.updatedID != "link-1" || paymentRepository.updatedStatus != model.PaymentLinkStatusPaid {
		t.Fatalf("payment link update = (%q, %q), want (link-1, %s)", paymentRepository.updatedID, paymentRepository.updatedStatus, model.PaymentLinkStatusPaid)
	}
	if tenantRepository.managerID != "manager-1" || tenantRepository.roomID != "room-1" {
		t.Fatalf("tenant lookup = (%q, %q), want (manager-1, room-1)", tenantRepository.managerID, tenantRepository.roomID)
	}
	if userRepository.userID != "manager-1" {
		t.Fatalf("manager lookup = %q, want manager-1", userRepository.userID)
	}
	if len(zaloService.sent) != 2 {
		t.Fatalf("sent notifications = %d, want 2", len(zaloService.sent))
	}
	if zaloService.sent[0].chatID != "tenant-zalo" || zaloService.sent[1].chatID != "manager-zalo" {
		t.Fatalf("notification chat IDs = %q, %q; want tenant-zalo, manager-zalo", zaloService.sent[0].chatID, zaloService.sent[1].chatID)
	}
}

// TestPayOSWebhookServiceHandleWebhookVerifiesChecksumKey covers PayOS signature validation.
func TestPayOSWebhookServiceHandleWebhookVerifiesChecksumKey(t *testing.T) {
	checksumKey := "checksum-secret"
	managerZaloID := "manager-zalo"
	webhookData := &payos.WebhookDataType{
		OrderCode:     123456,
		Amount:        150000,
		AccountNumber: "9704000012345678",
		Reference:     "PAYOS-REF",
	}
	signature, err := createPayOSSignature(webhookData, checksumKey)
	if err != nil {
		t.Fatalf("createPayOSSignature() error = %v", err)
	}
	success := true
	body, err := json.Marshal(payos.WebhookType{
		Success:   &success,
		Data:      webhookData,
		Signature: signature,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	paymentRepository := &fakePayOSPaymentRepository{
		paymentLink: &model.InvoicePaymentLink{ID: "link-1", InvoiceID: "invoice-1"},
	}
	invoiceRepository := &fakePayOSInvoiceRepository{
		invoice: &model.InvoiceWithRoom{
			Invoice: model.Invoice{
				ID:     "invoice-1",
				RoomID: "room-1",
				Period: "2024-01",
			},
			RoomName:  "A101",
			ManagerID: "manager-1",
		},
	}
	svc := &paymentService{
		paymentRepo: paymentRepository,
		invoiceRepo: invoiceRepository,
		tenantRepo:  &fakePayOSTenantRepository{},
		userRepo:    &fakePayOSUserRepository{user: &model.User{ID: "manager-1", ZaloUserID: &managerZaloID}},
		zaloService: &fakePayOSZaloService{},
		credentialService: fakePaymentCredentialService{credentials: map[string]string{
			"client_id":    "client-id",
			"api_key":      "api-key",
			"checksum_key": checksumKey,
		}},
		registry: NewPaymentProviderRegistry(NewPayOSProvider()),
	}

	if err := svc.HandleWebhook(context.Background(), model.PaymentProviderPayOS, "manager-1", body); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if paymentRepository.event == nil {
		t.Fatal("expected payment event to be recorded")
	}
	if invoiceRepository.updatedStatus != model.InvoiceStatusPaid {
		t.Fatalf("updated invoice status = %q, want %q", invoiceRepository.updatedStatus, model.InvoiceStatusPaid)
	}

	svc.credentialService = fakePaymentCredentialService{credentials: map[string]string{
		"client_id":    "client-id",
		"api_key":      "api-key",
		"checksum_key": "wrong-key",
	}}
	if err := svc.HandleWebhook(context.Background(), model.PaymentProviderPayOS, "manager-1", body); err == nil {
		t.Fatal("expected signature mismatch error, got nil")
	}
}

// TestPayOSWebhookServiceHandleWebhookDoesNotProcessOtherManagerLink prevents cross-manager payment updates.
func TestPayOSWebhookServiceHandleWebhookDoesNotProcessOtherManagerLink(t *testing.T) {
	checksumKey := "checksum-secret"
	webhookData := &payos.WebhookDataType{
		OrderCode:     123456,
		Amount:        150000,
		AccountNumber: "9704000012345678",
		Reference:     "PAYOS-REF",
	}
	signature, err := createPayOSSignature(webhookData, checksumKey)
	if err != nil {
		t.Fatalf("createPayOSSignature() error = %v", err)
	}
	success := true
	body, err := json.Marshal(payos.WebhookType{
		Success:   &success,
		Data:      webhookData,
		Signature: signature,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	paymentRepository := &fakePayOSPaymentRepository{
		paymentLink:   &model.InvoicePaymentLink{ID: "link-2", InvoiceID: "invoice-2"},
		linkManagerID: "manager-2",
	}
	invoiceRepository := &fakePayOSInvoiceRepository{}
	svc := &paymentService{
		paymentRepo: paymentRepository,
		invoiceRepo: invoiceRepository,
		tenantRepo:  &fakePayOSTenantRepository{},
		userRepo:    &fakePayOSUserRepository{},
		zaloService: &fakePayOSZaloService{},
		credentialService: fakePaymentCredentialService{credentials: map[string]string{
			"client_id":    "client-id",
			"api_key":      "api-key",
			"checksum_key": checksumKey,
		}},
		registry: NewPaymentProviderRegistry(NewPayOSProvider()),
	}

	if err := svc.HandleWebhook(context.Background(), model.PaymentProviderPayOS, "manager-1", body); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}
	if paymentRepository.event == nil {
		t.Fatal("expected unmatched payment event to be recorded")
	}
	if paymentRepository.event.Status != model.PaymentEventStatusUnmatched {
		t.Fatalf("payment event status = %q, want %q", paymentRepository.event.Status, model.PaymentEventStatusUnmatched)
	}
	if paymentRepository.event.InvoiceID != nil {
		t.Fatalf("payment event invoice ID = %v, want nil", *paymentRepository.event.InvoiceID)
	}
	if invoiceRepository.updatedID != "" {
		t.Fatalf("updated invoice ID = %q, want empty", invoiceRepository.updatedID)
	}
	if paymentRepository.updatedID != "" {
		t.Fatalf("updated payment link ID = %q, want empty", paymentRepository.updatedID)
	}
}
