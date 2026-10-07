package payment

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/pkg/sepay"
)

// sePayTestCredentials returns a full credential map for a given webhook auth method.
func sePayTestCredentials(authMethod string) map[string]string {
	return map[string]string{
		"bank_short_name":     "MBBank",
		"environment":         sepay.EnvironmentSandbox,
		"account_number":      "123456789",
		"account_name":        "NGUYEN VAN A",
		"code_prefix":         "PT",
		"webhook_auth_method": authMethod,
		"webhook_api_key":     "secret-api-key",
		"webhook_secret":      "secret-hmac",
	}
}

// TestSePayCreatePaymentLink verifies QR URL composition and payment code generation.
func TestSePayCreatePaymentLink(t *testing.T) {
	provider := NewSePayProvider()
	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{ID: "abc12345-def6-7890", TotalAmount: 500000},
	}

	link, err := provider.CreatePaymentLink(context.Background(), PaymentCreateInput{
		Invoice:     invoice,
		Credentials: sePayTestCredentials("apikey"),
	})
	if err != nil {
		t.Fatalf("CreatePaymentLink() error = %v", err)
	}

	if !strings.HasPrefix(link.ProviderOrderRef, "PT") {
		t.Errorf("payment code = %q, want prefix PT", link.ProviderOrderRef)
	}
	if link.Amount != 500000 {
		t.Errorf("amount = %d, want 500000", link.Amount)
	}

	parsed, err := url.Parse(link.QRCode)
	if err != nil {
		t.Fatalf("parse QR url: %v", err)
	}
	if got := parsed.Host; got != "qr.sepay.vn" {
		t.Errorf("QR host = %q, want qr.sepay.vn", got)
	}
	query := parsed.Query()
	if query.Get("acc") != "123456789" || query.Get("bank") != "MBBank" {
		t.Errorf("QR acc/bank = %q/%q", query.Get("acc"), query.Get("bank"))
	}
	if query.Get("amount") != "500000" {
		t.Errorf("QR amount = %q, want 500000", query.Get("amount"))
	}
	if query.Get("des") != link.ProviderOrderRef {
		t.Errorf("QR des = %q, want payment code %q", query.Get("des"), link.ProviderOrderRef)
	}
}

// TestSePayCreatePaymentLinkRejectsNonPositiveAmount ensures zero-amount invoices are refused.
func TestSePayCreatePaymentLinkRejectsNonPositiveAmount(t *testing.T) {
	provider := NewSePayProvider()
	invoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "abc123", TotalAmount: 0}}

	if _, err := provider.CreatePaymentLink(context.Background(), PaymentCreateInput{
		Invoice:     invoice,
		Credentials: sePayTestCredentials("apikey"),
	}); err == nil {
		t.Fatal("expected error for non-positive amount, got nil")
	}
}

// TestSePayCreatePaymentLinkRejectsMissingCredentials verifies QR generation requires bank config.
func TestSePayCreatePaymentLinkRejectsMissingCredentials(t *testing.T) {
	provider := NewSePayProvider()
	invoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "abc123", TotalAmount: 1000}}

	if _, err := provider.CreatePaymentLink(context.Background(), PaymentCreateInput{
		Invoice:     invoice,
		Credentials: map[string]string{"bank_short_name": "MBBank"},
	}); err == nil {
		t.Fatal("expected error for missing SePay credentials, got nil")
	}
}

// TestSePayCreatePaymentLinkRetriesOnCollision ensures a colliding code triggers a new attempt.
func TestSePayCreatePaymentLinkRetriesOnCollision(t *testing.T) {
	provider := NewSePayProvider()
	invoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "abc12345", TotalAmount: 1000}}

	var first string
	link, err := provider.CreatePaymentLink(context.Background(), PaymentCreateInput{
		Invoice:     invoice,
		Credentials: sePayTestCredentials("apikey"),
		ProviderOrderRefExists: func(_ context.Context, ref string) (bool, error) {
			if first == "" {
				first = ref
				return true, nil // force a collision on the first attempt
			}
			return false, nil
		},
	})
	if err != nil {
		t.Fatalf("CreatePaymentLink() error = %v", err)
	}
	if link.ProviderOrderRef == first {
		t.Errorf("expected a different code after collision, got same %q", link.ProviderOrderRef)
	}
}

// TestSePayCreatePaymentLinkReturnsCollisionErrors verifies repository lookup failures bubble up.
func TestSePayCreatePaymentLinkReturnsCollisionErrors(t *testing.T) {
	provider := NewSePayProvider()
	invoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "abc12345", TotalAmount: 1000}}
	checkErr := errors.New("lookup failed")

	if _, err := provider.CreatePaymentLink(context.Background(), PaymentCreateInput{
		Invoice:     invoice,
		Credentials: sePayTestCredentials("apikey"),
		ProviderOrderRefExists: func(context.Context, string) (bool, error) {
			return false, checkErr
		},
	}); !errors.Is(err, checkErr) {
		t.Fatalf("error = %v, want lookup error", err)
	}
}

// TestSePayCancelPaymentLinkIsNoop verifies SePay QR payments have no remote cancellation call.
func TestSePayCancelPaymentLinkIsNoop(t *testing.T) {
	if err := NewSePayProvider().CancelPaymentLink(context.Background(), PaymentCancelInput{}); err != nil {
		t.Fatalf("CancelPaymentLink() error = %v", err)
	}
}

// signSePayBody computes the SePay HMAC signature header value for a raw body.
func signSePayBody(secret, timestamp string, body []byte) string {
	return sepay.SignHMAC(secret, timestamp, body)
}

const sePayIncomingBody = `{"id":92704,"gateway":"Vietcombank","transactionDate":"2024-07-02 11:08:33","accountNumber":"123456789","subAccount":"","code":"PTABC123","content":"PTABC123 chuyen tien","transferType":"in","transferAmount":500000,"referenceCode":"FT24012345678"}`

// TestSePayVerifyWebhookAPIKey covers API-key authentication success and failure.
func TestSePayVerifyWebhookAPIKey(t *testing.T) {
	provider := NewSePayProvider()
	body := []byte(sePayIncomingBody)

	validHeaders := http.Header{}
	validHeaders.Set("Authorization", "Apikey secret-api-key")
	event, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body:        body,
		Headers:     validHeaders,
		Credentials: sePayTestCredentials("apikey"),
	})
	if err != nil {
		t.Fatalf("VerifyWebhook() valid apikey error = %v", err)
	}
	if event.ProviderOrderRef != "PTABC123" {
		t.Errorf("ProviderOrderRef = %q, want PTABC123", event.ProviderOrderRef)
	}
	if event.Amount != 500000 {
		t.Errorf("Amount = %d, want 500000", event.Amount)
	}
	if event.TransactionReference != "92704" {
		t.Errorf("TransactionReference = %q, want 92704 (SePay id)", event.TransactionReference)
	}

	wrongHeaders := http.Header{}
	wrongHeaders.Set("Authorization", "Apikey wrong-key")
	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body:        body,
		Headers:     wrongHeaders,
		Credentials: sePayTestCredentials("apikey"),
	}); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("wrong apikey error = %v, want ErrPaymentWebhookInvalid", err)
	}

	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body:        body,
		Headers:     http.Header{},
		Credentials: sePayTestCredentials("apikey"),
	}); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("missing apikey header error = %v, want ErrPaymentWebhookInvalid", err)
	}
}

// TestSePayVerifyWebhookHMAC covers HMAC-SHA256 authentication success and failure.
func TestSePayVerifyWebhookHMAC(t *testing.T) {
	provider := NewSePayProvider()
	body := []byte(sePayIncomingBody)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	validHeaders := http.Header{}
	validHeaders.Set("X-SePay-Timestamp", timestamp)
	validHeaders.Set("X-SePay-Signature", signSePayBody("secret-hmac", timestamp, body))
	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body:        body,
		Headers:     validHeaders,
		Credentials: sePayTestCredentials("hmac"),
	}); err != nil {
		t.Fatalf("VerifyWebhook() valid hmac error = %v", err)
	}

	tamperedHeaders := http.Header{}
	tamperedHeaders.Set("X-SePay-Timestamp", timestamp)
	tamperedHeaders.Set("X-SePay-Signature", signSePayBody("secret-hmac", timestamp, []byte(`{"id":1}`)))
	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body:        body,
		Headers:     tamperedHeaders,
		Credentials: sePayTestCredentials("hmac"),
	}); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("tampered hmac error = %v, want ErrPaymentWebhookInvalid", err)
	}
}

// TestSePayVerifyWebhookAuthErrors covers missing provider auth configuration.
func TestSePayVerifyWebhookAuthErrors(t *testing.T) {
	provider := NewSePayProvider()
	body := []byte(sePayIncomingBody)

	tests := []struct {
		name        string
		credentials map[string]string
		headers     http.Header
		want        error
	}{
		{
			name:        "unknown method",
			credentials: map[string]string{"webhook_auth_method": "basic"},
			headers:     http.Header{},
			want:        ErrPaymentWebhookInvalid,
		},
		{
			name:        "missing api key config",
			credentials: map[string]string{"webhook_auth_method": "apikey"},
			headers:     http.Header{"Authorization": []string{"Apikey secret-api-key"}},
			want:        ErrPaymentCredentialsNotFound,
		},
		{
			name:        "missing hmac config",
			credentials: map[string]string{"webhook_auth_method": "hmac"},
			headers:     http.Header{},
			want:        ErrPaymentCredentialsNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
				Body:        body,
				Headers:     tt.headers,
				Credentials: tt.credentials,
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

// TestSePayVerifyWebhookIgnoresOutgoingAndMissingCode covers the ignore policy.
func TestSePayVerifyWebhookIgnoresOutgoingAndMissingCode(t *testing.T) {
	provider := NewSePayProvider()
	creds := sePayTestCredentials("none")

	outgoing := []byte(`{"id":1,"transferType":"out","transferAmount":100,"code":"PTX"}`)
	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body: outgoing, Credentials: creds,
	}); !errors.Is(err, ErrPaymentWebhookIgnored) {
		t.Errorf("outgoing transfer error = %v, want ErrPaymentWebhookIgnored", err)
	}

	missingCode := []byte(`{"id":2,"transferType":"in","transferAmount":100,"code":""}`)
	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body: missingCode, Credentials: creds,
	}); !errors.Is(err, ErrPaymentWebhookIgnored) {
		t.Errorf("missing code error = %v, want ErrPaymentWebhookIgnored", err)
	}
}

// newSePayWebhookService wires a payment service with the SePay provider for HandleWebhook tests.
func newSePayWebhookService(link *model.InvoicePaymentLink) (*paymentService, *fakePayOSPaymentRepository, *fakePayOSInvoiceRepository) {
	paymentRepository := &fakePayOSPaymentRepository{paymentLink: link}
	invoiceRepository := &fakePayOSInvoiceRepository{
		invoice: &model.InvoiceWithRoom{
			Invoice:   model.Invoice{ID: "invoice-1", RoomID: "room-1"},
			ManagerID: "manager-1",
		},
	}
	svc := &paymentService{
		paymentRepo:       paymentRepository,
		invoiceRepo:       invoiceRepository,
		tenantRepo:        &fakePayOSTenantRepository{},
		userRepo:          &fakePayOSUserRepository{user: &model.User{ID: "manager-1"}},
		credentialService: fakePaymentCredentialService{credentials: sePayTestCredentials("none")},
		registry:          NewPaymentProviderRegistry(NewSePayProvider()),
	}
	return svc, paymentRepository, invoiceRepository
}

// TestSePayHandleWebhookUnderpaidDoesNotMarkPaid verifies the amount guard blocks underpaid settlements.
func TestSePayHandleWebhookUnderpaidDoesNotMarkPaid(t *testing.T) {
	svc, paymentRepository, invoiceRepository := newSePayWebhookService(
		&model.InvoicePaymentLink{ID: "link-1", InvoiceID: "invoice-1", ProviderOrderRef: "PTABC123", Amount: 999999},
	)

	body := []byte(`{"id":92704,"transferType":"in","transferAmount":500000,"code":"PTABC123","accountNumber":"123456789"}`)
	if err := svc.HandleWebhook(context.Background(), model.PaymentProviderSePay, "manager-1", body, nil); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}

	if invoiceRepository.updatedStatus == model.InvoiceStatusPaid {
		t.Error("invoice was marked PAID on an underpaid transaction")
	}
	if paymentRepository.event == nil {
		t.Error("expected the underpaid transaction to be recorded for audit")
	}
}

// TestSePayHandleWebhookSufficientMarksPaid verifies a full-amount transfer settles the invoice as SEPAY.
func TestSePayHandleWebhookSufficientMarksPaid(t *testing.T) {
	svc, _, invoiceRepository := newSePayWebhookService(
		&model.InvoicePaymentLink{ID: "link-1", InvoiceID: "invoice-1", ProviderOrderRef: "PTABC123", Amount: 500000},
	)

	body := []byte(`{"id":92704,"transferType":"in","transferAmount":500000,"code":"PTABC123","accountNumber":"123456789"}`)
	if err := svc.HandleWebhook(context.Background(), model.PaymentProviderSePay, "manager-1", body, nil); err != nil {
		t.Fatalf("HandleWebhook() error = %v", err)
	}

	if invoiceRepository.updatedStatus != model.InvoiceStatusPaid {
		t.Errorf("invoice status = %q, want %q", invoiceRepository.updatedStatus, model.InvoiceStatusPaid)
	}
	if invoiceRepository.updatedMethod != model.PaymentMethodSePay {
		t.Errorf("invoice method = %q, want %q", invoiceRepository.updatedMethod, model.PaymentMethodSePay)
	}
}

// fakeSePayLister serves pre-canned transaction pages keyed by requested page number.
type fakeSePayLister struct {
	pages map[int]*sepay.TransactionsResponse
	calls []sepay.ListTransactionsParams
	err   error
}

// ListTransactions records the request params and returns the configured page.
func (f *fakeSePayLister) ListTransactions(_ context.Context, _ string, params sepay.ListTransactionsParams) (*sepay.TransactionsResponse, error) {
	f.calls = append(f.calls, params)
	if f.err != nil {
		return nil, f.err
	}
	if resp, ok := f.pages[params.Page]; ok {
		return resp, nil
	}
	return &sepay.TransactionsResponse{}, nil
}

// fakeReconcileProcessor records every verified event it is asked to process.
type fakeReconcileProcessor struct {
	events []*VerifiedPaymentEvent
	err    error
}

// ProcessVerifiedTransaction stores the event for assertions.
func (f *fakeReconcileProcessor) ProcessVerifiedTransaction(_ context.Context, _, _ string, event *VerifiedPaymentEvent) error {
	f.events = append(f.events, event)
	return f.err
}

// pageWith builds a one-page response with the given transactions and has_more flag.
func pageWith(hasMore bool, txs ...sepay.Transaction) *sepay.TransactionsResponse {
	resp := &sepay.TransactionsResponse{Status: "success", Data: txs}
	resp.Meta.Pagination.HasMore = hasMore
	return resp
}

// TestSePayReconcileProcessesIncomingWithCodeAcrossPages verifies paging, filtering, and mapping.
func TestSePayReconcileProcessesIncomingWithCodeAcrossPages(t *testing.T) {
	lister := &fakeSePayLister{pages: map[int]*sepay.TransactionsResponse{
		1: pageWith(true,
			sepay.Transaction{ID: "uuid-in-1", Code: "PTAAA", AmountIn: 100000, TransferType: "in", AccountNumber: "123"},
			sepay.Transaction{ID: "uuid-out", Code: "PTOUT", AmountOut: 50000, TransferType: "out"},
			sepay.Transaction{ID: "uuid-nocode", Code: "", AmountIn: 70000, TransferType: "in"},
		),
		2: pageWith(false,
			sepay.Transaction{ID: "uuid-in-2", Code: "PTBBB", AmountIn: 200000, TransferType: "in", AccountNumber: "456"},
		),
	}}
	processor := &fakeReconcileProcessor{}
	creds := fakePaymentCredentialService{credentials: map[string]string{"api_token": "tok", "environment": sepay.EnvironmentSandbox}}

	svc := NewSePayReconciliationService(lister, creds, processor)
	svc.reconciler.Throttle = 0

	result, err := svc.ReconcileManager(context.Background(), "manager-1", "2026-06-01 00:00:00", "2026-06-27 00:00:00")
	if err != nil {
		t.Fatalf("ReconcileManager() error = %v", err)
	}

	if result.PagesFetched != 2 {
		t.Errorf("PagesFetched = %d, want 2", result.PagesFetched)
	}
	if result.Scanned != 4 {
		t.Errorf("Scanned = %d, want 4", result.Scanned)
	}
	if result.Processed != 2 {
		t.Errorf("Processed = %d, want 2 (only incoming with code)", result.Processed)
	}
	if len(processor.events) != 2 {
		t.Fatalf("processed events = %d, want 2", len(processor.events))
	}

	first := processor.events[0]
	if first.ProviderOrderRef != "PTAAA" || first.Amount != 100000 || first.TransactionReference != "uuid-in-1" {
		t.Errorf("first event = {%q, %d, %q}, want {PTAAA, 100000, uuid-in-1}", first.ProviderOrderRef, first.Amount, first.TransactionReference)
	}
	if processor.events[1].ProviderOrderRef != "PTBBB" {
		t.Errorf("second event ref = %q, want PTBBB", processor.events[1].ProviderOrderRef)
	}
	if lister.calls[0].Environment != sepay.EnvironmentSandbox {
		t.Errorf("environment = %q, want sandbox", lister.calls[0].Environment)
	}
}

// TestSePayReconcileRequiresAPIToken verifies a missing api_token is rejected.
func TestSePayReconcileRequiresAPIToken(t *testing.T) {
	svc := NewSePayReconciliationService(
		&fakeSePayLister{},
		fakePaymentCredentialService{credentials: map[string]string{}},
		&fakeReconcileProcessor{},
	)
	svc.reconciler.Throttle = 0

	if _, err := svc.ReconcileManager(context.Background(), "manager-1", "", ""); !errors.Is(err, ErrPaymentCredentialsNotFound) {
		t.Errorf("error = %v, want ErrPaymentCredentialsNotFound", err)
	}
}

// TestProcessVerifiedTransactionSkipsPaidLink verifies the cross-source guard: an already-paid
// link (settled by a webhook) is recorded for audit but not re-settled by reconciliation.
func TestProcessVerifiedTransactionSkipsPaidLink(t *testing.T) {
	svc, paymentRepository, invoiceRepository := newSePayWebhookService(
		&model.InvoicePaymentLink{ID: "link-1", InvoiceID: "invoice-1", ProviderOrderRef: "PTABC123", Amount: 500000, Status: model.PaymentLinkStatusPaid},
	)

	event := &VerifiedPaymentEvent{
		ProviderOrderRef:     "PTABC123",
		Amount:               500000,
		TransactionReference: "uuid-from-api-v2",
		MatchingMethod:       paymentMatchOrderRef,
		SignatureResult:      paymentSignatureValid,
	}
	if err := svc.ProcessVerifiedTransaction(context.Background(), model.PaymentProviderSePay, "manager-1", event); err != nil {
		t.Fatalf("ProcessVerifiedTransaction() error = %v", err)
	}

	if invoiceRepository.updatedStatus == model.InvoiceStatusPaid {
		t.Error("already-paid invoice was settled again by reconciliation")
	}
	if paymentRepository.event == nil {
		t.Error("expected the reconciled transaction to be recorded for audit")
	}
}
