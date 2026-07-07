package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

// errStub is a sentinel error used to assert error propagation in service tests.
var errStub = errors.New("stub failure")

// cfgPaymentRepo is a configurable paymentRepository stub with per-method error and value injection.
type cfgPaymentRepo struct {
	activeLink       *model.InvoicePaymentLink
	activeErr        error
	linkByRef        *model.InvoicePaymentLink
	linkByRefErr     error
	linkForManager   *model.InvoicePaymentLink
	linkForMgrErr    error
	createLinkErr    error
	updateStatusErr  error
	eventExists      bool
	eventExistsErr   error
	createEventErr   error
	createdLink      *model.InvoicePaymentLink
	createdEvent     *model.PaymentEvent
	updatedStatuses  map[string]string
}

// CreatePaymentLink records the persisted link and returns the configured error.
func (r *cfgPaymentRepo) CreatePaymentLink(_ context.Context, link *model.InvoicePaymentLink) error {
	r.createdLink = link
	if link != nil {
		link.ID = "new-link"
	}
	return r.createLinkErr
}

// GetActivePaymentLinkByProvider returns the configured active link/error.
func (r *cfgPaymentRepo) GetActivePaymentLinkByProvider(context.Context, string, string) (*model.InvoicePaymentLink, error) {
	return r.activeLink, r.activeErr
}

// GetPaymentLinkByProviderOrderRef returns the configured link/error for order-ref lookups.
func (r *cfgPaymentRepo) GetPaymentLinkByProviderOrderRef(context.Context, string, string) (*model.InvoicePaymentLink, error) {
	return r.linkByRef, r.linkByRefErr
}

// GetPaymentLinkByProviderOrderRefForManager returns the configured manager-scoped link/error.
func (r *cfgPaymentRepo) GetPaymentLinkByProviderOrderRefForManager(context.Context, string, string, string) (*model.InvoicePaymentLink, error) {
	return r.linkForManager, r.linkForMgrErr
}

// UpdatePaymentLinkStatus records the status transition and returns the configured error.
func (r *cfgPaymentRepo) UpdatePaymentLinkStatus(_ context.Context, id, status string) error {
	if r.updatedStatuses == nil {
		r.updatedStatuses = map[string]string{}
	}
	r.updatedStatuses[id] = status
	return r.updateStatusErr
}

// CheckProviderEventExists returns the configured idempotency result/error.
func (r *cfgPaymentRepo) CheckProviderEventExists(context.Context, string, string, string) (bool, error) {
	return r.eventExists, r.eventExistsErr
}

// CreatePaymentEvent records the audit event and returns the configured error.
func (r *cfgPaymentRepo) CreatePaymentEvent(_ context.Context, event *model.PaymentEvent) error {
	r.createdEvent = event
	return r.createEventErr
}

// cfgInvoiceRepo is a configurable paymentInvoiceRepository stub.
type cfgInvoiceRepo struct {
	invoice *model.InvoiceWithRoom
	err     error
}

// SystemUpdateInvoiceStatusAndMethod returns the configured invoice/error.
func (r *cfgInvoiceRepo) SystemUpdateInvoiceStatusAndMethod(context.Context, string, string, string) (*model.InvoiceWithRoom, error) {
	return r.invoice, r.err
}

// cfgTenantRepo is a configurable paymentTenantRepository stub.
type cfgTenantRepo struct {
	tenants []model.FullInfoTenant
	err     error
}

// ListTenantByRoomID returns the configured tenants/error.
func (r *cfgTenantRepo) ListTenantByRoomID(context.Context, string, string) ([]model.FullInfoTenant, error) {
	return r.tenants, r.err
}

// cfgUserRepo is a configurable paymentUserRepository stub.
type cfgUserRepo struct {
	user *model.User
	err  error
}

// GetByUserID returns the configured manager user/error.
func (r *cfgUserRepo) GetByUserID(context.Context, string) (*model.User, error) {
	return r.user, r.err
}

// cfgProvider is a configurable PaymentProvider stub with per-method injection.
type cfgProvider struct {
	name       string
	createLink *PaymentProviderLink
	createErr  error
	cancelErr  error
	verifyEvt  *VerifiedPaymentEvent
	verifyErr  error
}

// Name returns the configured provider key.
func (p *cfgProvider) Name() string { return p.name }

// CreatePaymentLink returns the configured provider link/error.
func (p *cfgProvider) CreatePaymentLink(context.Context, PaymentCreateInput) (*PaymentProviderLink, error) {
	return p.createLink, p.createErr
}

// CancelPaymentLink returns the configured cancel error.
func (p *cfgProvider) CancelPaymentLink(context.Context, PaymentCancelInput) error {
	return p.cancelErr
}

// VerifyWebhook returns the configured verified event/error.
func (p *cfgProvider) VerifyWebhook(context.Context, PaymentWebhookInput) (*VerifiedPaymentEvent, error) {
	return p.verifyEvt, p.verifyErr
}

// cfgCredentialService overrides credential resolution on top of the base fake.
type cfgCredentialService struct {
	fakePaymentCredentialService
	getErr          error
	preferred       string
	preferredErr    error
}

// GetCredentials returns the configured credentials or an injected error.
func (s cfgCredentialService) GetCredentials(context.Context, string, string) (map[string]string, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.credentials, nil
}

// ResolvePreferredProvider returns the configured preferred provider or an injected error.
func (s cfgCredentialService) ResolvePreferredProvider(context.Context, string) (string, error) {
	if s.preferredErr != nil {
		return "", s.preferredErr
	}
	return s.preferred, nil
}

// cfgZalo is a ZaloMessenger stub whose send result is configurable.
type cfgZalo struct {
	sendErr error
	sent    int
}

// SendTextMessage records the call and returns the configured error.
func (z *cfgZalo) SendTextMessage(context.Context, string, string, string) error {
	z.sent++
	return z.sendErr
}

// TestNewPaymentServiceTrimsAppURL verifies the constructor wires dependencies and trims the app URL slash.
func TestNewPaymentServiceTrimsAppURL(t *testing.T) {
	svc := NewPaymentService(
		&cfgPaymentRepo{}, &cfgInvoiceRepo{}, &cfgTenantRepo{}, &cfgUserRepo{},
		&cfgZalo{}, cfgCredentialService{}, NewPaymentProviderRegistry(), "https://app.example/",
	)
	concrete, ok := svc.(*paymentService)
	if !ok {
		t.Fatalf("NewPaymentService returned %T, want *paymentService", svc)
	}
	if concrete.appURL != "https://app.example" {
		t.Errorf("appURL = %q, want trailing slash trimmed", concrete.appURL)
	}
}

// TestSetZaloService attaches a messenger after construction.
func TestSetZaloService(t *testing.T) {
	svc := &paymentService{}
	messenger := &cfgZalo{}
	svc.SetZaloService(messenger)
	if svc.zaloService != messenger {
		t.Fatal("SetZaloService did not attach the messenger")
	}
}

// newTestService assembles a paymentService around the given provider and stub repos.
func newTestService(provider PaymentProvider, creds cfgCredentialService) (*paymentService, *cfgPaymentRepo, *cfgInvoiceRepo) {
	paymentRepo := &cfgPaymentRepo{}
	invoiceRepo := &cfgInvoiceRepo{invoice: &model.InvoiceWithRoom{
		Invoice:   model.Invoice{ID: "invoice-1", RoomID: "room-1", Period: "2024-01"},
		RoomName:  "A101",
		ManagerID: "manager-1",
	}}
	registry := NewPaymentProviderRegistry()
	if provider != nil {
		registry = NewPaymentProviderRegistry(provider)
	}
	svc := &paymentService{
		paymentRepo:       paymentRepo,
		invoiceRepo:       invoiceRepo,
		tenantRepo:        &cfgTenantRepo{},
		userRepo:          &cfgUserRepo{user: &model.User{ID: "manager-1"}},
		credentialService: creds,
		registry:          registry,
	}
	return svc, paymentRepo, invoiceRepo
}

// testInvoice returns a minimal invoice suitable for payment-link creation.
func testInvoice() *model.InvoiceWithRoom {
	return &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1", TotalAmount: 150000}}
}

// TestCreatePaymentLinkForInvoiceUnknownProvider verifies an unregistered provider errors.
func TestCreatePaymentLinkForInvoiceUnknownProvider(t *testing.T) {
	svc, _, _ := newTestService(nil, cfgCredentialService{})
	if _, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "ghost", testInvoice(), "T"); !errors.Is(err, ErrPaymentProviderNotFound) {
		t.Fatalf("error = %v, want ErrPaymentProviderNotFound", err)
	}
}

// TestCreatePaymentLinkForInvoiceCredentialError propagates credential lookup failures.
func TestCreatePaymentLinkForInvoiceCredentialError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, _, _ := newTestService(provider, cfgCredentialService{getErr: errStub})
	if _, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "sepay", testInvoice(), "T"); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestCreatePaymentLinkForInvoiceActiveLookupError wraps a failed active-link lookup.
func TestCreatePaymentLinkForInvoiceActiveLookupError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	paymentRepo.activeErr = errStub
	if _, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "sepay", testInvoice(), "T"); err == nil {
		t.Fatal("expected error from active link lookup, got nil")
	}
}

// TestCreatePaymentLinkForInvoiceStaleUpdateError surfaces a mark-stale failure.
func TestCreatePaymentLinkForInvoiceStaleUpdateError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	paymentRepo.activeLink = &model.InvoicePaymentLink{ID: "old", Amount: 1}
	paymentRepo.updateStatusErr = errStub
	if _, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "sepay", testInvoice(), "T"); err == nil {
		t.Fatal("expected error from mark stale, got nil")
	}
}

// TestCreatePaymentLinkForInvoiceProviderCreateError propagates a provider create failure.
func TestCreatePaymentLinkForInvoiceProviderCreateError(t *testing.T) {
	provider := &cfgProvider{name: "sepay", createErr: errStub}
	svc, _, _ := newTestService(provider, cfgCredentialService{})
	if _, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "sepay", testInvoice(), "T"); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestCreatePaymentLinkForInvoiceSaveError surfaces a DB persistence failure.
func TestCreatePaymentLinkForInvoiceSaveError(t *testing.T) {
	provider := &cfgProvider{name: "sepay", createLink: &PaymentProviderLink{Amount: 150000}}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	paymentRepo.createLinkErr = errStub
	if _, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "sepay", testInvoice(), "T"); err == nil {
		t.Fatal("expected error from save, got nil")
	}
}

// TestCreatePaymentLinkForInvoiceSuccess persists and returns a freshly created link.
func TestCreatePaymentLinkForInvoiceSuccess(t *testing.T) {
	provider := &cfgProvider{name: "sepay", createLink: &PaymentProviderLink{
		ProviderOrderRef: "PT1", PaymentLinkID: "pl-1", CheckoutURL: "u", QRCode: "q", Amount: 150000,
	}}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	link, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "sepay", testInvoice(), "Tenant")
	if err != nil {
		t.Fatalf("CreatePaymentLinkForInvoice() error = %v", err)
	}
	if link == nil || link.ID != "new-link" || link.Status != model.PaymentLinkStatusActive {
		t.Fatalf("link = %#v, want persisted active link", link)
	}
	if paymentRepo.createdLink == nil {
		t.Fatal("expected the link to be persisted")
	}
}

// TestCreatePreferredPaymentLinkResolvesProvider routes creation through the resolved provider.
func TestCreatePreferredPaymentLinkResolvesProvider(t *testing.T) {
	provider := &cfgProvider{name: "sepay", createLink: &PaymentProviderLink{Amount: 150000}}
	svc, _, _ := newTestService(provider, cfgCredentialService{preferred: "sepay"})
	if _, err := svc.CreatePreferredPaymentLinkForInvoice(context.Background(), "manager-1", testInvoice(), "T"); err != nil {
		t.Fatalf("CreatePreferredPaymentLinkForInvoice() error = %v", err)
	}
}

// TestCreatePreferredPaymentLinkResolveError propagates preferred-provider resolution failures.
func TestCreatePreferredPaymentLinkResolveError(t *testing.T) {
	svc, _, _ := newTestService(nil, cfgCredentialService{preferredErr: errStub})
	if _, err := svc.CreatePreferredPaymentLinkForInvoice(context.Background(), "manager-1", testInvoice(), "T"); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestCancelPaymentLinkUnknownProvider verifies an unregistered provider errors.
func TestCancelPaymentLinkUnknownProvider(t *testing.T) {
	svc, _, _ := newTestService(nil, cfgCredentialService{})
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "ghost", "ref", "reason"); !errors.Is(err, ErrPaymentProviderNotFound) {
		t.Fatalf("error = %v, want ErrPaymentProviderNotFound", err)
	}
}

// TestCancelPaymentLinkCredentialError propagates credential lookup failures.
func TestCancelPaymentLinkCredentialError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, _, _ := newTestService(provider, cfgCredentialService{getErr: errStub})
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "sepay", "ref", "reason"); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestCancelPaymentLinkProviderError propagates a provider cancel failure.
func TestCancelPaymentLinkProviderError(t *testing.T) {
	provider := &cfgProvider{name: "sepay", cancelErr: errStub}
	svc, _, _ := newTestService(provider, cfgCredentialService{})
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "sepay", "ref", "reason"); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestCancelPaymentLinkNoLocalLink returns nil when no local link matches.
func TestCancelPaymentLinkNoLocalLink(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, _, _ := newTestService(provider, cfgCredentialService{})
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "sepay", "ref", "reason"); err != nil {
		t.Fatalf("CancelPaymentLink() error = %v, want nil", err)
	}
}

// TestCancelPaymentLinkLookupError propagates the local link lookup failure.
func TestCancelPaymentLinkLookupError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	paymentRepo.linkForMgrErr = errStub
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "sepay", "ref", "reason"); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestCancelPaymentLinkMarksCancelled updates a matched local link to cancelled.
func TestCancelPaymentLinkMarksCancelled(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	paymentRepo.linkForManager = &model.InvoicePaymentLink{ID: "link-1"}
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "sepay", "ref", "reason"); err != nil {
		t.Fatalf("CancelPaymentLink() error = %v", err)
	}
	if paymentRepo.updatedStatuses["link-1"] != model.PaymentLinkStatusCancelled {
		t.Fatalf("status = %q, want CANCELLED", paymentRepo.updatedStatuses["link-1"])
	}
}

// TestCancelPaymentLinkStatusUpdateError surfaces a cancelled-status persistence failure.
func TestCancelPaymentLinkStatusUpdateError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, paymentRepo, _ := newTestService(provider, cfgCredentialService{})
	paymentRepo.linkForManager = &model.InvoicePaymentLink{ID: "link-1"}
	paymentRepo.updateStatusErr = errStub
	if err := svc.CancelPaymentLink(context.Background(), "manager-1", "sepay", "ref", "reason"); err == nil {
		t.Fatal("expected error from cancelled status update, got nil")
	}
}

// TestHandleWebhookUnknownProvider verifies an unregistered provider errors.
func TestHandleWebhookUnknownProvider(t *testing.T) {
	svc, _, _ := newTestService(nil, cfgCredentialService{})
	if err := svc.HandleWebhook(context.Background(), "ghost", "manager-1", nil, nil); !errors.Is(err, ErrPaymentProviderNotFound) {
		t.Fatalf("error = %v, want ErrPaymentProviderNotFound", err)
	}
}

// TestHandleWebhookCredentialError propagates credential lookup failures.
func TestHandleWebhookCredentialError(t *testing.T) {
	provider := &cfgProvider{name: "sepay"}
	svc, _, _ := newTestService(provider, cfgCredentialService{getErr: errStub})
	if err := svc.HandleWebhook(context.Background(), "sepay", "manager-1", nil, nil); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestHandleWebhookIgnoredEvent swallows the ignore sentinel and returns nil.
func TestHandleWebhookIgnoredEvent(t *testing.T) {
	provider := &cfgProvider{name: "sepay", verifyErr: ErrPaymentWebhookIgnored}
	svc, _, _ := newTestService(provider, cfgCredentialService{})
	if err := svc.HandleWebhook(context.Background(), "sepay", "manager-1", nil, nil); err != nil {
		t.Fatalf("HandleWebhook() error = %v, want nil for ignored", err)
	}
}

// TestHandleWebhookVerifyError propagates a signature verification failure.
func TestHandleWebhookVerifyError(t *testing.T) {
	provider := &cfgProvider{name: "sepay", verifyErr: errStub}
	svc, _, _ := newTestService(provider, cfgCredentialService{})
	if err := svc.HandleWebhook(context.Background(), "sepay", "manager-1", nil, nil); !errors.Is(err, errStub) {
		t.Fatalf("error = %v, want errStub", err)
	}
}

// TestHandleWebhookNilEvent maps a nil verified event to the nil-data error.
func TestHandleWebhookNilEvent(t *testing.T) {
	provider := &cfgProvider{name: "sepay"} // verifyEvt nil, verifyErr nil
	svc, _, _ := newTestService(provider, cfgCredentialService{})
	if err := svc.HandleWebhook(context.Background(), "sepay", "manager-1", nil, nil); !errors.Is(err, ErrPayOSVerifiedDataNil) {
		t.Fatalf("error = %v, want ErrPayOSVerifiedDataNil", err)
	}
}

// newVerifiedEvent returns a minimal verified event for ProcessVerifiedTransaction tests.
func newVerifiedEvent() *VerifiedPaymentEvent {
	return &VerifiedPaymentEvent{ProviderOrderRef: "PT1", Amount: 150000, TransactionReference: "tx-1"}
}

// TestProcessVerifiedTransactionEventExistsCheckError wraps a failed idempotency check.
func TestProcessVerifiedTransactionEventExistsCheckError(t *testing.T) {
	svc, paymentRepo, _ := newTestService(&cfgProvider{name: "sepay"}, cfgCredentialService{})
	paymentRepo.eventExistsErr = errStub
	if err := svc.ProcessVerifiedTransaction(context.Background(), "sepay", "manager-1", newVerifiedEvent()); err == nil {
		t.Fatal("expected error from event-exists check, got nil")
	}
}

// TestProcessVerifiedTransactionDuplicate returns nil when the event already exists.
func TestProcessVerifiedTransactionDuplicate(t *testing.T) {
	svc, paymentRepo, _ := newTestService(&cfgProvider{name: "sepay"}, cfgCredentialService{})
	paymentRepo.eventExists = true
	if err := svc.ProcessVerifiedTransaction(context.Background(), "sepay", "manager-1", newVerifiedEvent()); err != nil {
		t.Fatalf("ProcessVerifiedTransaction() error = %v, want nil for duplicate", err)
	}
	if paymentRepo.createdEvent != nil {
		t.Fatal("duplicate event must not be recorded again")
	}
}

// TestProcessVerifiedTransactionLinkLookupError wraps a failed payment-link lookup.
func TestProcessVerifiedTransactionLinkLookupError(t *testing.T) {
	svc, paymentRepo, _ := newTestService(&cfgProvider{name: "sepay"}, cfgCredentialService{})
	paymentRepo.linkForMgrErr = errStub
	if err := svc.ProcessVerifiedTransaction(context.Background(), "sepay", "manager-1", newVerifiedEvent()); err == nil {
		t.Fatal("expected error from link lookup, got nil")
	}
}

// TestProcessVerifiedTransactionCreateEventError wraps a failed audit-event write.
func TestProcessVerifiedTransactionCreateEventError(t *testing.T) {
	svc, paymentRepo, _ := newTestService(&cfgProvider{name: "sepay"}, cfgCredentialService{})
	paymentRepo.createEventErr = errStub
	if err := svc.ProcessVerifiedTransaction(context.Background(), "sepay", "manager-1", newVerifiedEvent()); err == nil {
		t.Fatal("expected error from create event, got nil")
	}
}

// TestProcessVerifiedTransactionUnmatched records an unmatched event without settling an invoice.
func TestProcessVerifiedTransactionUnmatched(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newTestService(&cfgProvider{name: "sepay"}, cfgCredentialService{})
	// linkForManager stays nil -> unmatched transaction.
	if err := svc.ProcessVerifiedTransaction(context.Background(), "sepay", "manager-1", newVerifiedEvent()); err != nil {
		t.Fatalf("ProcessVerifiedTransaction() error = %v", err)
	}
	if paymentRepo.createdEvent == nil || paymentRepo.createdEvent.Status != model.PaymentEventStatusUnmatched {
		t.Fatal("expected an unmatched audit event")
	}
	if invoiceRepo.err != nil {
		t.Fatal("invoice repo should not have been consulted with an error")
	}
}

// TestProcessInvoicePaymentUpdateError wraps a failed invoice status update.
func TestProcessInvoicePaymentUpdateError(t *testing.T) {
	svc, _, invoiceRepo := newTestService(&cfgProvider{name: "sepay"}, cfgCredentialService{})
	invoiceRepo.err = errStub
	if err := svc.processInvoicePayment(context.Background(), "sepay", "invoice-1", 150000, nil); err == nil {
		t.Fatal("expected error from invoice update, got nil")
	}
}

// TestProcessInvoicePaymentTolerantOfSideFailures logs but does not fail on link/tenant/manager errors.
func TestProcessInvoicePaymentTolerantOfSideFailures(t *testing.T) {
	paymentRepo := &cfgPaymentRepo{updateStatusErr: errStub}
	invoiceRepo := &cfgInvoiceRepo{invoice: &model.InvoiceWithRoom{
		Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1", Period: "2024-01"}, RoomName: "A101", ManagerID: "manager-1",
	}}
	svc := &paymentService{
		paymentRepo: paymentRepo,
		invoiceRepo: invoiceRepo,
		tenantRepo:  &cfgTenantRepo{err: errStub},
		userRepo:    &cfgUserRepo{err: errStub},
		zaloService: &cfgZalo{},
	}
	if err := svc.processInvoicePayment(context.Background(), "sepay", "invoice-1", 150000, &model.InvoicePaymentLink{ID: "link-1"}); err != nil {
		t.Fatalf("processInvoicePayment() error = %v, want nil despite side failures", err)
	}
}

// TestProcessInvoicePaymentNotifiesManager sends the manager notification when a Zalo ID exists.
func TestProcessInvoicePaymentNotifiesManager(t *testing.T) {
	zaloID := "mgr-zalo"
	zalo := &cfgZalo{sendErr: errStub} // exercise the best-effort error log branch too
	svc := &paymentService{
		paymentRepo: &cfgPaymentRepo{},
		invoiceRepo: &cfgInvoiceRepo{invoice: &model.InvoiceWithRoom{
			Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1", Period: "2024-01"}, RoomName: "A101", ManagerID: "manager-1",
		}},
		tenantRepo:  &cfgTenantRepo{tenants: []model.FullInfoTenant{{FullName: "A", ZaloUserID: "t-zalo"}}},
		userRepo:    &cfgUserRepo{user: &model.User{ID: "manager-1", ZaloUserID: &zaloID}},
		zaloService: zalo,
	}
	if err := svc.processInvoicePayment(context.Background(), "sepay", "invoice-1", 150000, &model.InvoicePaymentLink{ID: "link-1"}); err != nil {
		t.Fatalf("processInvoicePayment() error = %v", err)
	}
	if zalo.sent != 2 {
		t.Fatalf("sent = %d, want 2 (tenant + manager)", zalo.sent)
	}
}

// TestSendTextMessageBestEffortNilMessenger returns quietly when no messenger is configured.
func TestSendTextMessageBestEffortNilMessenger(t *testing.T) {
	svc := &paymentService{}
	svc.sendTextMessageBestEffort(context.Background(), "manager-1", "chat", "hi") // must not panic
}

// TestNormalizePaymentProvider covers the empty-default and lowercasing branches.
func TestNormalizePaymentProvider(t *testing.T) {
	if got := normalizePaymentProvider(""); got != model.PaymentProviderPayOS {
		t.Errorf("empty provider = %q, want %q", got, model.PaymentProviderPayOS)
	}
	if got := normalizePaymentProvider("SePay"); got != "sepay" {
		t.Errorf("SePay = %q, want sepay", got)
	}
}

// TestPaymentMethodForProvider covers the known providers and the uppercase default.
func TestPaymentMethodForProvider(t *testing.T) {
	if got := paymentMethodForProvider(model.PaymentProviderPayOS); got != model.PaymentMethodPayOS {
		t.Errorf("payos method = %q, want %q", got, model.PaymentMethodPayOS)
	}
	if got := paymentMethodForProvider(model.PaymentProviderSePay); got != model.PaymentMethodSePay {
		t.Errorf("sepay method = %q, want %q", got, model.PaymentMethodSePay)
	}
	if got := paymentMethodForProvider("momo"); got != "MOMO" {
		t.Errorf("default method = %q, want MOMO", got)
	}
}

// TestPaymentProviderRegistrySkipsNilAndMissing covers the nil-provider skip and nil-registry Get.
func TestPaymentProviderRegistrySkipsNilAndMissing(t *testing.T) {
	registry := NewPaymentProviderRegistry(nil, &cfgProvider{name: "sepay"})
	if _, ok := registry.Get("sepay"); !ok {
		t.Fatal("expected sepay provider to be registered")
	}
	if _, ok := registry.Get("missing"); ok {
		t.Fatal("expected missing provider lookup to fail")
	}
	var nilRegistry *PaymentProviderRegistry
	if _, ok := nilRegistry.Get("sepay"); ok {
		t.Fatal("expected nil registry Get to return false")
	}
}
