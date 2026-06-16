package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

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
		paymentRepo: repository.NewInvoicePaymentRepository(bunDB),
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
