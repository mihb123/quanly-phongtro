package payment

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	invoicerepo "github.com/mihb123/quanly-phongtro/internal/repository/invoice"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

type fakePaymentProvider struct {
	name string
	link *PaymentProviderLink
}

// Name returns the provider key used by the payment registry.
func (p fakePaymentProvider) Name() string {
	return p.name
}

// CreatePaymentLink returns the configured provider link without external calls.
func (p fakePaymentProvider) CreatePaymentLink(context.Context, PaymentCreateInput) (*PaymentProviderLink, error) {
	return p.link, nil
}

// CancelPaymentLink satisfies PaymentProvider for tests that do not cancel.
func (p fakePaymentProvider) CancelPaymentLink(context.Context, PaymentCancelInput) error {
	return nil
}

// VerifyWebhook satisfies PaymentProvider for tests that do not process webhooks.
func (p fakePaymentProvider) VerifyWebhook(context.Context, PaymentWebhookInput) (*VerifiedPaymentEvent, error) {
	return nil, nil
}

// setupPayOSServiceTestDB creates a mocked Bun database for PayOS service tests.
func setupPayOSServiceTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	return bun.NewDB(sqlDB, pgdialect.New()), mock
}

// TestBuildOrderReferenceCode verifies the invoice-room code format used in PayOS descriptions.
func TestBuildOrderReferenceCode(t *testing.T) {
	got := buildOrderReferenceCode("e5tyabcd", "90urzzzz")
	if got != "e5ty-90ur" {
		t.Fatalf("buildOrderReferenceCode() = %q, want e5ty-90ur", got)
	}
}

// TestPayOSOrderCodeUsesReferenceCode keeps the first PayOS order code tied to the readable code.
func TestPayOSOrderCodeUsesReferenceCode(t *testing.T) {
	got := payOSOrderCode("e5ty-90ur", "invoice-id", "room-id", 0)
	want := base36OrderCode("e5ty-90ur")
	if got != want {
		t.Fatalf("payOSOrderCode() = %d, want %d", got, want)
	}
	if got <= 0 || got > maxPayOSOrderCode {
		t.Fatalf("payOSOrderCode() = %d, want within PayOS range", got)
	}
}

// TestPaymentServiceCreatePaymentLinkReusesActiveLink avoids creating duplicate active provider links.
func TestPaymentServiceCreatePaymentLinkReusesActiveLink(t *testing.T) {
	bunDB, mock := setupPayOSServiceTestDB(t)
	defer bunDB.Close()

	mock.ExpectQuery(`.*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"invoice_id",
			"order_code",
			"payment_link_id",
			"checkout_url",
			"qr_code",
			"amount",
			"status",
		}).AddRow(
			"link-1",
			"invoice-1",
			12345,
			"payos-link-1",
			"https://checkout.example",
			"qr-code",
			150000,
			model.PaymentLinkStatusActive,
		))

	svc := &paymentService{
		paymentRepo: invoicerepo.NewInvoicePaymentRepository(bunDB),
		registry:    NewPaymentProviderRegistry(NewPayOSProvider()),
		credentialService: fakePaymentCredentialService{credentials: map[string]string{
			"client_id":    "client-id",
			"api_key":      "api-key",
			"checksum_key": "checksum-key",
		}},
	}
	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:          "invoice-1",
			RoomID:      "room-1",
			TotalAmount: 150000,
			Status:      model.InvoiceStatusUnpaid,
		},
	}

	paymentLink, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", model.PaymentProviderPayOS, invoice, "Tenant")
	if err != nil {
		t.Fatalf("CreatePaymentLinkForInvoice() error = %v", err)
	}
	if paymentLink == nil || paymentLink.ID != "link-1" {
		t.Fatalf("CreatePaymentLinkForInvoice() link = %#v, want link-1", paymentLink)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

// TestPaymentServiceCreatePaymentLinkRefreshesStaleAmount verifies stale active links are not reused.
func TestPaymentServiceCreatePaymentLinkRefreshesStaleAmount(t *testing.T) {
	bunDB, mock := setupPayOSServiceTestDB(t)
	defer bunDB.Close()

	mock.ExpectQuery(`.*`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"invoice_id",
			"provider",
			"provider_order_ref",
			"order_code",
			"payment_link_id",
			"checkout_url",
			"qr_code",
			"amount",
			"status",
		}).AddRow(
			"link-1",
			"invoice-1",
			"fakepay",
			"old-ref",
			12345,
			"old-link",
			"https://old.example",
			"old-qr",
			100000,
			model.PaymentLinkStatusActive,
		))
	mock.ExpectExec(`UPDATE "invoice_payment_links"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO "invoice_payment_links"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("link-2", time.Now(), time.Now()))

	svc := &paymentService{
		paymentRepo: invoicerepo.NewInvoicePaymentRepository(bunDB),
		registry: NewPaymentProviderRegistry(fakePaymentProvider{
			name: "fakepay",
			link: &PaymentProviderLink{
				ProviderOrderRef: "new-ref",
				OrderCode:        67890,
				PaymentLinkID:    "new-link",
				CheckoutURL:      "https://new.example",
				QRCode:           "new-qr",
				Amount:           150000,
			},
		}),
		credentialService: fakePaymentCredentialService{credentials: map[string]string{
			"client_id":    "client-id",
			"api_key":      "api-key",
			"checksum_key": "checksum-key",
		}},
	}
	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:          "invoice-1",
			RoomID:      "room-1",
			TotalAmount: 150000,
			Status:      model.InvoiceStatusUnpaid,
		},
	}

	paymentLink, err := svc.CreatePaymentLinkForInvoice(context.Background(), "manager-1", "fakepay", invoice, "Tenant")
	if err != nil {
		t.Fatalf("CreatePaymentLinkForInvoice() error = %v", err)
	}
	if paymentLink == nil || paymentLink.ID != "link-2" || paymentLink.Amount != 150000 {
		t.Fatalf("CreatePaymentLinkForInvoice() link = %#v, want new 150000 link", paymentLink)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
