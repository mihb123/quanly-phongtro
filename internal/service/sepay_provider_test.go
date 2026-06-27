package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

// sePayTestCredentials returns a full credential map for a given webhook auth method.
func sePayTestCredentials(authMethod string) map[string]string {
	return map[string]string{
		"bank_short_name":     "MBBank",
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

// signSePayBody computes the SePay HMAC signature header value for a raw body.
func signSePayBody(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp + "." + string(body)))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
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
	const timestamp = "1719893313"

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
