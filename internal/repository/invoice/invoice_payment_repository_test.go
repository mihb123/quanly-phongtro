package invoice_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/repository/invoice"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func setupPaymentTestDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	sqldb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	db := bun.NewDB(sqldb, pgdialect.New())
	return db, mock
}

func TestInvoicePaymentRepository_CreatePaymentLink(t *testing.T) {
	bunDB, mock := setupPaymentTestDB(t)
	defer bunDB.Close()

	repo := invoice.NewInvoicePaymentRepository(bunDB)
	ctx := context.Background()

	paymentLink := &model.InvoicePaymentLink{
		InvoiceID:        "inv-1",
		Provider:         model.PaymentProviderSePay,
		ProviderOrderRef: "PHINV1",
		Amount:           100000,
		CheckoutURL:      "http://checkout.url",
		QRCode:           "qr-code",
		PaymentLinkID:    "pay-link-id",
		Status:           model.PaymentLinkStatusActive,
	}
	createdAt := time.Date(2026, 10, 8, 1, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`^INSERT INTO "invoice_payment_links" \("invoice_id", "provider", "provider_order_ref", "order_code", "payment_link_id", "checkout_url", "qr_code", "amount", "status"\) VALUES \('inv-1', 'sepay', 'PHINV1', 0, 'pay-link-id', 'http://checkout.url', 'qr-code', 100000, 'ACTIVE'\) RETURNING id, created_at, updated_at$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("019e0000-0000-7000-8000-000000000001", createdAt, createdAt))

	err := repo.CreatePaymentLink(ctx, paymentLink)
	if err != nil {
		t.Fatalf("CreatePaymentLink() error = %v", err)
	}
	if paymentLink.ID != "019e0000-0000-7000-8000-000000000001" {
		t.Errorf("CreatePaymentLink() ID = %q, want DB-generated id", paymentLink.ID)
	}
	if !paymentLink.CreatedAt.Equal(createdAt) || !paymentLink.UpdatedAt.Equal(createdAt) {
		t.Errorf("CreatePaymentLink() timestamps = %v/%v, want %v", paymentLink.CreatedAt, paymentLink.UpdatedAt, createdAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestInvoicePaymentRepository_GetActivePaymentLink(t *testing.T) {
	bunDB, mock := setupPaymentTestDB(t)
	defer bunDB.Close()

	repo := invoice.NewInvoicePaymentRepository(bunDB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "invoice_id"}).AddRow("link-1", "inv-1"))

		link, err := repo.GetActivePaymentLink(ctx, "inv-1")
		if err != nil {
			t.Errorf("GetActivePaymentLink() error = %v", err)
		}
		if link == nil || link.ID != "link-1" {
			t.Errorf("expected link-1, got %v", link)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnError(sql.ErrNoRows)

		link, err := repo.GetActivePaymentLink(ctx, "inv-1")
		if err != nil {
			t.Errorf("GetActivePaymentLink() error = %v", err)
		}
		if link != nil {
			t.Errorf("expected nil link, got %v", link)
		}
	})
}

func TestInvoicePaymentRepository_UpdatePaymentLinkStatus(t *testing.T) {
	bunDB, mock := setupPaymentTestDB(t)
	defer bunDB.Close()

	repo := invoice.NewInvoicePaymentRepository(bunDB)
	ctx := context.Background()

	mock.ExpectExec(`UPDATE .*invoice_payment_link.*`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.UpdatePaymentLinkStatus(ctx, "link-1", "PAID")
	if err != nil {
		t.Errorf("UpdatePaymentLinkStatus() error = %v", err)
	}
}

func TestInvoicePaymentRepository_GetPaymentLinkByOrderCode(t *testing.T) {
	bunDB, mock := setupPaymentTestDB(t)
	defer bunDB.Close()

	repo := invoice.NewInvoicePaymentRepository(bunDB)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "order_code"}).AddRow("link-1", 12345))

		link, err := repo.GetPaymentLinkByOrderCode(ctx, 12345)
		if err != nil {
			t.Errorf("GetPaymentLinkByOrderCode() error = %v", err)
		}
		if link == nil || link.ID != "link-1" {
			t.Errorf("expected link-1, got %v", link)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnError(sql.ErrNoRows)

		link, err := repo.GetPaymentLinkByOrderCode(ctx, 12345)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if link != nil {
			t.Errorf("expected nil link, got %v", link)
		}
	})
}

func TestInvoicePaymentRepository_CreatePaymentEvent(t *testing.T) {
	bunDB, mock := setupPaymentTestDB(t)
	defer bunDB.Close()

	repo := invoice.NewInvoicePaymentRepository(bunDB)
	ctx := context.Background()
	createdAt := time.Date(2026, 7, 7, 20, 4, 58, 0, time.UTC)
	managerID := "4ab21578-3071-4c45-85cb-53b75cff6f98"

	event := &model.PayOSPaymentEvent{
		Provider:             model.PaymentProviderSePay,
		ManagerID:            &managerID,
		ProviderOrderRef:     "PT123",
		OrderCode:            12345,
		Amount:               100000,
		TransactionReference: "ref-1",
		RawPayload:           "{}",
		SignatureResult:      "VALID",
		Status:               model.PaymentEventStatusProcessed,
	}

	mock.ExpectQuery(`INSERT INTO "payment_events" \("provider", "manager_id"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("event-1", createdAt))

	err := repo.CreatePaymentEvent(ctx, event)
	if err != nil {
		t.Errorf("CreatePaymentEvent() error = %v", err)
	}
	if event.ID != "event-1" {
		t.Errorf("expected generated event ID, got %q", event.ID)
	}
}

func TestInvoicePaymentRepository_CheckEventExists(t *testing.T) {
	bunDB, mock := setupPaymentTestDB(t)
	defer bunDB.Close()

	repo := invoice.NewInvoicePaymentRepository(bunDB)
	ctx := context.Background()

	t.Run("exists", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		exists, err := repo.CheckEventExists(ctx, 12345, "ref-1")
		if err != nil {
			t.Errorf("CheckEventExists() error = %v", err)
		}
		if !exists {
			t.Errorf("expected true, got false")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		exists, err := repo.CheckEventExists(ctx, 12345, "ref-2")
		if err != nil {
			t.Errorf("CheckEventExists() error = %v", err)
		}
		if exists {
			t.Errorf("expected false, got true")
		}
	})

	t.Run("db error", func(t *testing.T) {
		mock.ExpectQuery(`.*`).
			WillReturnError(errors.New("db error"))

		exists, err := repo.CheckEventExists(ctx, 12345, "ref-3")
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if exists {
			t.Errorf("expected false on error, got true")
		}
	})
}
