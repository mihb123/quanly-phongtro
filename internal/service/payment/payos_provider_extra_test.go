package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/payOSHQ/payos-lib-golang/v2"
)

// payosCredentials returns a complete PayOS credential map for provider tests.
func payosCredentials() map[string]string {
	return map[string]string{"client_id": "cid", "api_key": "key", "checksum_key": "checksum-secret"}
}

// payosRoundTripFunc adapts a function to an http.RoundTripper for stubbing the PayOS SDK transport.
type payosRoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip delegates to the wrapped function.
func (f payosRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// payosSignedResponse builds a Code "00" PayOS envelope with a valid body signature over data.
func payosSignedResponse(data map[string]any, checksumKey string) *http.Response {
	signature, _ := createPayOSSignature(data, checksumKey)
	body, _ := json.Marshal(map[string]any{
		"code":      "00",
		"desc":      "success",
		"data":      data,
		"signature": signature,
	})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

// TestNewPayOSClient covers the missing-credentials error and the successful client build.
func TestNewPayOSClient(t *testing.T) {
	if _, err := newPayOSClient(map[string]string{"client_id": "cid"}); err == nil {
		t.Fatal("expected error for incomplete PayOS credentials, got nil")
	}
	client, err := newPayOSClient(payosCredentials())
	if err != nil {
		t.Fatalf("newPayOSClient() error = %v", err)
	}
	if client == nil {
		t.Fatal("newPayOSClient() returned nil client")
	}
}

// TestPayOSDescription covers the short description and the 25-character truncation.
func TestPayOSDescription(t *testing.T) {
	if got := payOSDescription("e5ty-90ur", "A101"); got != "HD e5ty-90ur PA101" {
		t.Errorf("short description = %q", got)
	}
	long := payOSDescription("e5ty-90ur", "A-very-long-room-name-here")
	if len(long) != 25 {
		t.Errorf("truncated description len = %d, want 25", len(long))
	}
}

// TestOrderCodePart covers the base36 filtering and the empty fallback.
func TestOrderCodePart(t *testing.T) {
	if got := orderCodePart("A1$b2c3d4"); got != "a1b2" {
		t.Errorf("orderCodePart = %q, want a1b2", got)
	}
	if got := orderCodePart("---"); got != "0" {
		t.Errorf("orderCodePart empty = %q, want 0", got)
	}
}

// TestBase36OrderCodeOverflow verifies an over-long reference collapses to 0.
func TestBase36OrderCodeOverflow(t *testing.T) {
	if got := base36OrderCode("zzzzzzzzzzzzzzz"); got != 0 {
		t.Errorf("overflow base36OrderCode = %d, want 0", got)
	}
}

// TestPayOSOrderCodeFallsBackToHash covers the hash branches for empty and retried codes.
func TestPayOSOrderCodeFallsBackToHash(t *testing.T) {
	empty := payOSOrderCode("", "invoice", "room", 0)
	if empty <= 0 || empty > maxPayOSOrderCode {
		t.Errorf("hashed order code = %d, want within PayOS range", empty)
	}
	retry := payOSOrderCode("e5ty-90ur", "invoice", "room", 1)
	if retry <= 0 || retry > maxPayOSOrderCode {
		t.Errorf("retry order code = %d, want within PayOS range", retry)
	}
}

// TestGenerateUniquePayOSOrderCode covers the nil-checker, unique, error, and exhaustion paths.
func TestGenerateUniquePayOSOrderCode(t *testing.T) {
	invoice := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1"}}
	ctx := context.Background()

	if _, err := generateUniquePayOSOrderCode(ctx, invoice, "e5ty-90ur", nil); err != nil {
		t.Fatalf("nil checker error = %v", err)
	}
	if _, err := generateUniquePayOSOrderCode(ctx, invoice, "e5ty-90ur", func(context.Context, string) (bool, error) {
		return false, nil
	}); err != nil {
		t.Fatalf("unique checker error = %v", err)
	}
	if _, err := generateUniquePayOSOrderCode(ctx, invoice, "e5ty-90ur", func(context.Context, string) (bool, error) {
		return false, errStub
	}); err == nil {
		t.Fatal("expected error from checker, got nil")
	}
	if _, err := generateUniquePayOSOrderCode(ctx, invoice, "e5ty-90ur", func(context.Context, string) (bool, error) {
		return true, nil
	}); err == nil {
		t.Fatal("expected exhaustion error, got nil")
	}
}

// TestVerifyPayOSWebhookData covers the credential, nil-data, signature, mismatch, and success paths.
func TestVerifyPayOSWebhookData(t *testing.T) {
	const key = "checksum-secret"
	data := &payos.WebhookDataType{OrderCode: 123456, Amount: 150000, AccountNumber: "970400", Reference: "REF"}
	signature, err := createPayOSSignature(data, key)
	if err != nil {
		t.Fatalf("createPayOSSignature() error = %v", err)
	}

	if _, err := verifyPayOSWebhookData(payos.WebhookType{Data: data, Signature: signature}, ""); !errors.Is(err, ErrPaymentCredentialsNotFound) {
		t.Errorf("missing key error = %v, want ErrPaymentCredentialsNotFound", err)
	}
	if _, err := verifyPayOSWebhookData(payos.WebhookType{Data: nil, Signature: signature}, key); !errors.Is(err, ErrPayOSVerifiedDataNil) {
		t.Errorf("nil data error = %v, want ErrPayOSVerifiedDataNil", err)
	}
	if _, err := verifyPayOSWebhookData(payos.WebhookType{Data: data, Signature: ""}, key); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("missing signature error = %v, want ErrPaymentWebhookInvalid", err)
	}
	if _, err := verifyPayOSWebhookData(payos.WebhookType{Data: data, Signature: "deadbeef"}, key); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("mismatch error = %v, want ErrPaymentWebhookInvalid", err)
	}
	verified, err := verifyPayOSWebhookData(payos.WebhookType{Data: data, Signature: signature}, key)
	if err != nil || verified == nil {
		t.Fatalf("valid verify = (%v, %v), want data and nil error", verified, err)
	}
}

// TestIsPayOSTestPing covers the ping, real-order, and non-success cases.
func TestIsPayOSTestPing(t *testing.T) {
	yes := true
	no := false
	if !isPayOSTestPing(payos.WebhookType{Success: &yes, Data: &payos.WebhookDataType{OrderCode: 0}}) {
		t.Error("expected zero-order success to be a test ping")
	}
	if isPayOSTestPing(payos.WebhookType{Success: &yes, Data: &payos.WebhookDataType{OrderCode: 5}}) {
		t.Error("real order must not be a test ping")
	}
	if isPayOSTestPing(payos.WebhookType{Success: &no}) {
		t.Error("non-success must not be a test ping")
	}
}

// TestPayOSVerifyWebhookErrors covers the malformed-body and bad-signature branches of VerifyWebhook.
func TestPayOSVerifyWebhookErrors(t *testing.T) {
	provider := NewPayOSProvider()

	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{Body: []byte("{not json")}); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("bad body error = %v, want ErrPaymentWebhookInvalid", err)
	}

	success := true
	body, _ := json.Marshal(payos.WebhookType{
		Success:   &success,
		Data:      &payos.WebhookDataType{OrderCode: 5, Amount: 10},
		Signature: "deadbeef",
	})
	if _, err := provider.VerifyWebhook(context.Background(), PaymentWebhookInput{
		Body:        body,
		Credentials: payosCredentials(),
	}); !errors.Is(err, ErrPaymentWebhookInvalid) {
		t.Errorf("bad signature error = %v, want ErrPaymentWebhookInvalid", err)
	}
}

// TestCreatePayOSSignatureErrors covers the marshal and non-object unmarshal error branches.
func TestCreatePayOSSignatureErrors(t *testing.T) {
	if _, err := createPayOSSignature(make(chan int), "key"); err == nil {
		t.Error("expected marshal error for channel value, got nil")
	}
	if _, err := sortPayOSObjectByKey(42); err == nil {
		t.Error("expected unmarshal error for non-object value, got nil")
	}
	// A mixed-type object exercises the nil, string, number, and default value branches.
	if _, err := sortPayOSObjectByKey(map[string]any{"a": nil, "b": "x", "c": 3.5, "d": true}); err != nil {
		t.Errorf("mixed object signature error = %v", err)
	}
}

// TestPayOSCreatePaymentLinkErrors covers credential, amount, and order-code exhaustion failures.
func TestPayOSCreatePaymentLinkErrors(t *testing.T) {
	provider := NewPayOSProvider()
	ctx := context.Background()

	// Missing credentials -> client build fails.
	if _, err := provider.CreatePaymentLink(ctx, PaymentCreateInput{
		Invoice:     &model.InvoiceWithRoom{Invoice: model.Invoice{TotalAmount: 1000}},
		Credentials: map[string]string{},
	}); err == nil {
		t.Error("expected credential error, got nil")
	}

	// Non-positive amount is rejected.
	if _, err := provider.CreatePaymentLink(ctx, PaymentCreateInput{
		Invoice:     &model.InvoiceWithRoom{Invoice: model.Invoice{TotalAmount: 0}},
		Credentials: payosCredentials(),
	}); err == nil {
		t.Error("expected non-positive amount error, got nil")
	}

	// Order-code generation exhausts when every candidate already exists.
	if _, err := provider.CreatePaymentLink(ctx, PaymentCreateInput{
		Invoice:     &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1", TotalAmount: 1000}},
		Credentials: payosCredentials(),
		ProviderOrderRefExists: func(context.Context, string) (bool, error) {
			return true, nil
		},
	}); err == nil {
		t.Error("expected order-code exhaustion error, got nil")
	}
}

// TestPayOSCreatePaymentLinkSuccess drives the SDK against a stubbed transport returning a signed link.
func TestPayOSCreatePaymentLinkSuccess(t *testing.T) {
	const key = "checksum-secret"
	original := http.DefaultTransport
	http.DefaultTransport = payosRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		data := map[string]any{
			"bin":           "970400",
			"accountNumber": "12345",
			"accountName":   "SHOP",
			"amount":        1000,
			"description":   "HD",
			"orderCode":     999,
			"currency":      "VND",
			"paymentLinkId": "pl-123",
			"status":        "PENDING",
			"checkoutUrl":   "https://pay.example/checkout",
			"qrCode":        "qr-data",
		}
		return payosSignedResponse(data, key), nil
	})
	defer func() { http.DefaultTransport = original }()

	provider := NewPayOSProvider()
	link, err := provider.CreatePaymentLink(context.Background(), PaymentCreateInput{
		Invoice:     &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "invoice-1", RoomID: "room-1", TotalAmount: 1000}, RoomName: "A101"},
		TenantName:  "Nguyen Van A",
		AppURL:      "https://app.example",
		Credentials: map[string]string{"client_id": "cid", "api_key": "key", "checksum_key": key},
	})
	if err != nil {
		t.Fatalf("CreatePaymentLink() error = %v", err)
	}
	if link.PaymentLinkID != "pl-123" || link.CheckoutURL != "https://pay.example/checkout" || link.Amount != 1000 {
		t.Fatalf("link = %#v, want stubbed PayOS link", link)
	}
}

// TestPayOSCancelPaymentLink covers the credential, bad-ref, and successful-cancel paths.
func TestPayOSCancelPaymentLink(t *testing.T) {
	provider := NewPayOSProvider()
	ctx := context.Background()

	if err := provider.CancelPaymentLink(ctx, PaymentCancelInput{Credentials: map[string]string{}, ProviderOrderRef: "1"}); err == nil {
		t.Error("expected credential error, got nil")
	}
	if err := provider.CancelPaymentLink(ctx, PaymentCancelInput{Credentials: payosCredentials(), ProviderOrderRef: "not-a-number"}); err == nil {
		t.Error("expected parse error for bad order ref, got nil")
	}

	const key = "checksum-secret"
	original := http.DefaultTransport
	http.DefaultTransport = payosRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasSuffix(r.URL.Path, "/cancel") {
			t.Errorf("cancel called wrong path %q", r.URL.Path)
		}
		data := map[string]any{
			"id":        "pl-123",
			"orderCode": 999,
			"amount":    1000,
			"status":    "CANCELLED",
			"createdAt": "2024-01-01T00:00:00Z",
		}
		return payosSignedResponse(data, key), nil
	})
	defer func() { http.DefaultTransport = original }()

	if err := provider.CancelPaymentLink(ctx, PaymentCancelInput{
		Credentials:      map[string]string{"client_id": "cid", "api_key": "key", "checksum_key": key},
		ProviderOrderRef: "999",
		Reason:           "user cancelled",
	}); err != nil {
		t.Fatalf("CancelPaymentLink() error = %v", err)
	}
}
