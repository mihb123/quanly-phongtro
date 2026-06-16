package repository_test

import (
        "context"
        "database/sql"
        "errors"
        "testing"

        "github.com/DATA-DOG/go-sqlmock"
        "github.com/mihb123/quanly-phongtro/internal/model"
        "github.com/mihb123/quanly-phongtro/internal/repository"
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

        repo := repository.NewInvoicePaymentRepository(bunDB)
        ctx := context.Background()

        paymentLink := &model.InvoicePaymentLink{
                ID:            "link-1",
                InvoiceID:     "inv-1",
                OrderCode:     12345,
                Amount:        100000,
                CheckoutURL:   "http://checkout.url",
                QRCode:        "qr-code",
                PaymentLinkID: "pay-link-id",
                Status:        model.PaymentLinkStatusActive,
        }

        mock.ExpectQuery(`.*`).
                WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("link-1"))

        err := repo.CreatePaymentLink(ctx, paymentLink)
        if err != nil {
                t.Errorf("CreatePaymentLink() error = %v", err)
        }

        if err := mock.ExpectationsWereMet(); err != nil {
                t.Errorf("there were unfulfilled expectations: %s", err)
        }
}

func TestInvoicePaymentRepository_GetActivePaymentLink(t *testing.T) {
        bunDB, mock := setupPaymentTestDB(t)
        defer bunDB.Close()

        repo := repository.NewInvoicePaymentRepository(bunDB)
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

        repo := repository.NewInvoicePaymentRepository(bunDB)
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

        repo := repository.NewInvoicePaymentRepository(bunDB)
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

        repo := repository.NewInvoicePaymentRepository(bunDB)
        ctx := context.Background()

        event := &model.PayOSPaymentEvent{
                ID:                   "event-1",
                OrderCode:            12345,
                Amount:               100000,
                TransactionReference: "ref-1",
                Status:               model.PaymentEventStatusProcessed,
        }

        mock.ExpectQuery(`.*`).
                WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("event-1"))

        err := repo.CreatePaymentEvent(ctx, event)
        if err != nil {
                t.Errorf("CreatePaymentEvent() error = %v", err)
        }
}

func TestInvoicePaymentRepository_CheckEventExists(t *testing.T) {
        bunDB, mock := setupPaymentTestDB(t)
        defer bunDB.Close()

        repo := repository.NewInvoicePaymentRepository(bunDB)
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
